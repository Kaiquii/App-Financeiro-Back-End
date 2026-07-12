package auth

import (
	"testing"
	"time"
)

func TestLoginRateLimiterBlocksAfterFiveFailures(t *testing.T) {
	limiter := newLoginRateLimiter()
	now := time.Date(2026, time.July, 12, 12, 0, 0, 0, time.UTC)

	for i := 0; i < maxLoginFailures; i++ {
		if limiter.isBlocked("user@example.com", "192.0.2.10", now) {
			t.Fatalf("attempt %d should not be blocked before the limit", i+1)
		}
		limiter.recordFailure("user@example.com", "192.0.2.10", now)
	}

	if !limiter.isBlocked("user@example.com", "192.0.2.10", now) {
		t.Fatal("login should be blocked after five failures")
	}

	if limiter.isBlocked("user@example.com", "192.0.2.11", now) {
		t.Fatal("a different IP should not be blocked")
	}
}

func TestLoginRateLimiterExpiresAndClearsFailures(t *testing.T) {
	limiter := newLoginRateLimiter()
	now := time.Date(2026, time.July, 12, 12, 0, 0, 0, time.UTC)

	for i := 0; i < maxLoginFailures; i++ {
		limiter.recordFailure("user@example.com", "192.0.2.10", now)
	}

	if limiter.isBlocked("user@example.com", "192.0.2.10", now.Add(loginBlockDuration)) {
		t.Fatal("login should be allowed after the block duration")
	}

	limiter.recordFailure("user@example.com", "192.0.2.10", now.Add(loginBlockDuration))
	if limiter.isBlocked("user@example.com", "192.0.2.10", now.Add(loginBlockDuration)) {
		t.Fatal("a single failure after the block expires should not block login")
	}
}

func TestLoginRateLimiterClearRemovesFailures(t *testing.T) {
	limiter := newLoginRateLimiter()
	now := time.Date(2026, time.July, 12, 12, 0, 0, 0, time.UTC)

	for i := 0; i < maxLoginFailures-1; i++ {
		limiter.recordFailure("user@example.com", "192.0.2.10", now)
	}

	limiter.clear("user@example.com", "192.0.2.10")
	limiter.recordFailure("user@example.com", "192.0.2.10", now)

	if limiter.isBlocked("user@example.com", "192.0.2.10", now) {
		t.Fatal("a successful login should clear previous failures")
	}
}
