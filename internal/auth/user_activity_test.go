package auth

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestUserJSONIncludesLastActiveAt(t *testing.T) {
	lastActiveAt := time.Date(2026, time.August, 30, 14, 25, 36, 0, time.UTC)
	data, err := json.Marshal(User{ID: 1, Name: "Usuario", LastActiveAt: lastActiveAt})
	if err != nil {
		t.Fatalf("failed to marshal user: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatalf("failed to unmarshal user response: %v", err)
	}

	if got := response["last_active_at"]; got != "2026-08-30T14:25:36Z" {
		t.Fatalf("unexpected last_active_at: %v", got)
	}
}

func TestUserActivityBufferKeepsLatestTimestamp(t *testing.T) {
	buffer := newUserActivityBuffer()
	latest := time.Date(2026, time.August, 30, 14, 25, 36, 0, time.UTC)

	buffer.touch(42, latest)
	buffer.touch(42, latest.Add(-time.Minute))

	drained := buffer.drain()
	if got := drained[42]; !got.Equal(latest) {
		t.Fatalf("expected latest timestamp %v, got %v", latest, got)
	}
	if remaining := buffer.drain(); len(remaining) != 0 {
		t.Fatalf("expected empty buffer after drain, got %d entries", len(remaining))
	}
}

func TestUserActivityBufferRestoreDoesNotReplaceNewerTimestamp(t *testing.T) {
	buffer := newUserActivityBuffer()
	older := time.Date(2026, time.August, 30, 14, 25, 0, 0, time.UTC)
	newer := older.Add(time.Minute)

	buffer.touch(42, newer)
	buffer.restore(map[uint]time.Time{42: older, 7: older})

	drained := buffer.drain()
	if got := drained[42]; !got.Equal(newer) {
		t.Fatalf("expected newer timestamp %v, got %v", newer, got)
	}
	if got := drained[7]; !got.Equal(older) {
		t.Fatalf("expected restored timestamp %v, got %v", older, got)
	}
}

func TestUserActivityBufferSupportsConcurrentTouches(t *testing.T) {
	buffer := newUserActivityBuffer()
	base := time.Date(2026, time.August, 30, 14, 25, 0, 0, time.UTC)

	var waitGroup sync.WaitGroup
	for index := 0; index < 100; index++ {
		waitGroup.Add(1)
		go func(offset int) {
			defer waitGroup.Done()
			buffer.touch(42, base.Add(time.Duration(offset)*time.Second))
		}(index)
	}
	waitGroup.Wait()

	expected := base.Add(99 * time.Second)
	if got := buffer.drain()[42]; !got.Equal(expected) {
		t.Fatalf("expected latest concurrent timestamp %v, got %v", expected, got)
	}
}

func TestBuildUserActivityUpdateUsesSingleBatchStatement(t *testing.T) {
	first := time.Date(2026, time.August, 30, 14, 25, 0, 0, time.UTC)
	second := first.Add(time.Second)
	query, args := buildUserActivityUpdate(
		[]uint{7, 42},
		map[uint]time.Time{7: first, 42: second},
	)

	if strings.Count(query, "(CAST(? AS BIGINT), CAST(? AS TIMESTAMPTZ))") != 2 {
		t.Fatalf("expected two value tuples, got query %q", query)
	}
	if !strings.Contains(query, "GREATEST(users.last_active_at, activity.last_active_at)") {
		t.Fatalf("expected monotonic timestamp update, got query %q", query)
	}
	if len(args) != 4 || args[0] != uint(7) || args[2] != uint(42) {
		t.Fatalf("unexpected query arguments: %#v", args)
	}
}
