package auth

import (
	"sync"
	"time"
)

const (
	loginFailureWindow = 10 * time.Minute
	loginBlockDuration = 5 * time.Minute
	maxLoginFailures   = 5
)

type loginAttempt struct {
	failures     []time.Time
	blockedUntil time.Time
}

type loginRateLimiter struct {
	mu       sync.Mutex
	attempts map[string]loginAttempt
}

func newLoginRateLimiter() *loginRateLimiter {
	return &loginRateLimiter{attempts: make(map[string]loginAttempt)}
}

func (l *loginRateLimiter) isBlocked(email string, ipAddress string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	key := loginAttemptKey(email, ipAddress)
	attempt, exists := l.attempts[key]
	if !exists {
		return false
	}

	if attempt.blockedUntil.After(now) {
		return true
	}

	attempt.blockedUntil = time.Time{}
	attempt.failures = recentLoginFailures(attempt.failures, now)
	if len(attempt.failures) == 0 {
		delete(l.attempts, key)
		return false
	}

	l.attempts[key] = attempt
	return false
}

func (l *loginRateLimiter) recordFailure(email string, ipAddress string, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()

	key := loginAttemptKey(email, ipAddress)
	attempt := l.attempts[key]
	attempt.failures = append(recentLoginFailures(attempt.failures, now), now)

	if len(attempt.failures) >= maxLoginFailures {
		attempt.failures = nil
		attempt.blockedUntil = now.Add(loginBlockDuration)
	}

	l.attempts[key] = attempt
}

func (l *loginRateLimiter) clear(email string, ipAddress string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.attempts, loginAttemptKey(email, ipAddress))
}

func loginAttemptKey(email string, ipAddress string) string {
	return email + "\x00" + ipAddress
}

func recentLoginFailures(failures []time.Time, now time.Time) []time.Time {
	cutoff := now.Add(-loginFailureWindow)
	recent := failures[:0]
	for _, failure := range failures {
		if failure.After(cutoff) {
			recent = append(recent, failure)
		}
	}

	return recent
}

var loginAttempts = newLoginRateLimiter()
