package handler

import (
	"crypto/sha256"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	passwordAttemptLimit  = 8
	passwordAttemptWindow = 15 * time.Minute
)

type passwordAttempt struct {
	failures int
	resetAt  time.Time
}

type passwordAttemptLimiter struct {
	mu       sync.Mutex
	attempts map[[sha256.Size]byte]passwordAttempt
}

func newPasswordAttemptLimiter() *passwordAttemptLimiter {
	return &passwordAttemptLimiter{attempts: make(map[[sha256.Size]byte]passwordAttempt)}
}

func (l *passwordAttemptLimiter) allow(r *http.Request, email string, now time.Time) bool {
	if l == nil {
		return true
	}
	key := passwordAttemptKey(r, email)
	l.mu.Lock()
	defer l.mu.Unlock()
	attempt, exists := l.attempts[key]
	if !exists || !now.Before(attempt.resetAt) {
		delete(l.attempts, key)
		return true
	}
	return attempt.failures < passwordAttemptLimit
}

func (l *passwordAttemptLimiter) failure(r *http.Request, email string, now time.Time) {
	if l == nil {
		return
	}
	key := passwordAttemptKey(r, email)
	l.mu.Lock()
	defer l.mu.Unlock()
	attempt := l.attempts[key]
	if !now.Before(attempt.resetAt) {
		attempt = passwordAttempt{resetAt: now.Add(passwordAttemptWindow)}
	}
	attempt.failures++
	l.attempts[key] = attempt
	if len(l.attempts) > 10000 {
		for key, value := range l.attempts {
			if !now.Before(value.resetAt) {
				delete(l.attempts, key)
			}
		}
	}
}

func (l *passwordAttemptLimiter) success(r *http.Request, email string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	delete(l.attempts, passwordAttemptKey(r, email))
	l.mu.Unlock()
}

func passwordAttemptKey(r *http.Request, email string) [sha256.Size]byte {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return sha256.Sum256([]byte(host + "\x00" + strings.ToLower(strings.TrimSpace(email))))
}
