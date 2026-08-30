package auth

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

const (
	userActivityBatchSize    = 500
	userActivityFlushTimeout = 10 * time.Second
)

var pendingUserActivity = newUserActivityBuffer()

type userActivityBuffer struct {
	mu      sync.Mutex
	pending map[uint]time.Time
}

func newUserActivityBuffer() *userActivityBuffer {
	return &userActivityBuffer{pending: make(map[uint]time.Time)}
}

func (b *userActivityBuffer) touch(userID uint, at time.Time) {
	if userID == 0 {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	at = at.UTC()
	if current, exists := b.pending[userID]; !exists || at.After(current) {
		b.pending[userID] = at
	}
}

func (b *userActivityBuffer) drain() map[uint]time.Time {
	b.mu.Lock()
	defer b.mu.Unlock()

	drained := b.pending
	b.pending = make(map[uint]time.Time)
	return drained
}

func (b *userActivityBuffer) restore(activities map[uint]time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for userID, at := range activities {
		if current, exists := b.pending[userID]; !exists || at.After(current) {
			b.pending[userID] = at
		}
	}
}

func recordUserActivity(userID uint) {
	pendingUserActivity.touch(userID, time.Now())
}

func StartUserActivityTracking(ctx context.Context, db *gorm.DB, interval time.Duration) <-chan error {
	done := make(chan error, 1)
	go func() {
		defer close(done)
		if db == nil {
			done <- fmt.Errorf("conexao com o banco nao inicializada para registrar atividade")
			return
		}
		if interval <= 0 {
			done <- fmt.Errorf("intervalo de atividade deve ser maior que zero")
			return
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				flushContext, cancel := context.WithTimeout(context.Background(), userActivityFlushTimeout)
				err := flushUserActivity(flushContext, db, pendingUserActivity)
				cancel()
				if err != nil {
					log.Printf("Falha ao persistir atividade dos usuarios: %v", err)
				}
			case <-ctx.Done():
				flushContext, cancel := context.WithTimeout(context.Background(), userActivityFlushTimeout)
				done <- flushUserActivity(flushContext, db, pendingUserActivity)
				cancel()
				return
			}
		}
	}()
	return done
}

func flushUserActivity(ctx context.Context, db *gorm.DB, buffer *userActivityBuffer) error {
	activities := buffer.drain()
	if len(activities) == 0 {
		return nil
	}

	if err := persistUserActivity(ctx, db, activities); err != nil {
		buffer.restore(activities)
		return err
	}
	return nil
}

func persistUserActivity(ctx context.Context, db *gorm.DB, activities map[uint]time.Time) error {
	userIDs := make([]uint, 0, len(activities))
	for userID := range activities {
		userIDs = append(userIDs, userID)
	}
	sort.Slice(userIDs, func(i, j int) bool { return userIDs[i] < userIDs[j] })

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for start := 0; start < len(userIDs); start += userActivityBatchSize {
			end := min(start+userActivityBatchSize, len(userIDs))
			query, args := buildUserActivityUpdate(userIDs[start:end], activities)
			if err := tx.Exec(query, args...).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func buildUserActivityUpdate(userIDs []uint, activities map[uint]time.Time) (string, []interface{}) {
	var query strings.Builder
	query.WriteString(`UPDATE users AS users
SET last_active_at = GREATEST(users.last_active_at, activity.last_active_at)
FROM (VALUES `)

	args := make([]interface{}, 0, len(userIDs)*2)
	for index, userID := range userIDs {
		if index > 0 {
			query.WriteString(", ")
		}
		query.WriteString("(CAST(? AS BIGINT), CAST(? AS TIMESTAMPTZ))")
		args = append(args, userID, activities[userID])
	}

	query.WriteString(`) AS activity(user_id, last_active_at)
WHERE users.id = activity.user_id`)
	return query.String(), args
}
