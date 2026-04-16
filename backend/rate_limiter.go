package main

import (
	"sync"
	"time"
)

// LoginRateLimiter implements a simple sliding window rate limiter for login attempts
type LoginRateLimiter struct {
	mu          sync.Mutex
	attempts    map[string]*LoginAttempt
	maxAttempts int
	window      time.Duration
	banDuration time.Duration
}

// LoginAttempt tracks login attempts for an IP address
type LoginAttempt struct {
	Count        int
	FirstAttempt time.Time
	LastAttempt  time.Time
	BannedUntil  time.Time
}

// NewLoginRateLimiter creates a new rate limiter
// maxAttempts: maximum failed attempts before ban
// window: time window for counting attempts
// banDuration: how long to ban after exceeding max attempts
func NewLoginRateLimiter(maxAttempts int, window, banDuration time.Duration) *LoginRateLimiter {
	limiter := &LoginRateLimiter{
		attempts:    make(map[string]*LoginAttempt),
		maxAttempts: maxAttempts,
		window:      window,
		banDuration: banDuration,
	}

	// Start cleanup goroutine
	go limiter.cleanup()

	return limiter
}

// Allow checks if an IP address is allowed to attempt login
func (l *LoginRateLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.cleanupExpired()

	attempt, exists := l.attempts[ip]
	if !exists {
		return true
	}

	// Check if currently banned
	if time.Now().Before(attempt.BannedUntil) {
		return false
	}

	// Check if window has expired, reset
	if time.Since(attempt.FirstAttempt) > l.window {
		attempt.Count = 0
		attempt.FirstAttempt = time.Now()
	}

	return attempt.Count < l.maxAttempts
}

// RecordFailure records a failed login attempt
func (l *LoginRateLimiter) RecordFailure(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	attempt, exists := l.attempts[ip]
	if !exists {
		l.attempts[ip] = &LoginAttempt{
			Count:        1,
			FirstAttempt: time.Now(),
			LastAttempt:  time.Now(),
		}
		return
	}

	attempt.Count++
	attempt.LastAttempt = time.Now()

	// Ban if exceeded max attempts
	if attempt.Count >= l.maxAttempts {
		attempt.BannedUntil = time.Now().Add(l.banDuration)
	}
}

// RecordSuccess records a successful login and resets the counter
func (l *LoginRateLimiter) RecordSuccess(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.attempts, ip)
}

// cleanupExpired removes expired entries from the map
func (l *LoginRateLimiter) cleanupExpired() {
	now := time.Now()
	for ip, attempt := range l.attempts {
		// Remove if ban has expired and no recent attempts
		if now.After(attempt.BannedUntil) && now.Sub(attempt.LastAttempt) > l.window {
			delete(l.attempts, ip)
		}
	}
}

// cleanup runs periodically to clean up expired entries
func (l *LoginRateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		l.mu.Lock()
		l.cleanupExpired()
		l.mu.Unlock()
	}
}
