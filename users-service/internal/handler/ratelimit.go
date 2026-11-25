package handler

import (
	"strconv"
	"sync"
	"time"
)

type bucket struct {
	count   int
	resetAt time.Time
}

type rateLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	hits   map[string]*bucket
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{limit: limit, window: window, hits: make(map[string]*bucket)}
}

func (l *rateLimiter) allow(key string) (bool, time.Duration) {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.hits) > 10000 {
		for k, b := range l.hits {
			if now.After(b.resetAt) {
				delete(l.hits, k)
			}
		}
	}

	b, ok := l.hits[key]
	if !ok || now.After(b.resetAt) {
		l.hits[key] = &bucket{count: 1, resetAt: now.Add(l.window)}
		return true, 0
	}
	if b.count >= l.limit {
		return false, b.resetAt.Sub(now)
	}
	b.count++
	return true, 0
}

func retryAfterSeconds(d time.Duration) string {
	return strconv.Itoa(int(d.Seconds()) + 1)
}

var sensitivePaths = map[string]bool{
	"/api/users/register":           true,
	"/api/users/confirm":            true,
	"/api/users/login":              true,
	"/api/users/login/otp":          true,
	"/api/users/password/forgot":    true,
	"/api/users/password/reset":     true,
	"/api/users/magic-link/request": true,
	"/api/users/magic-link/login":   true,
}
