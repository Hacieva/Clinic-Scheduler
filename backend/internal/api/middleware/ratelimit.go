package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"time"
)

// RateLimiter tracks request counts per IP using a sliding window.
type RateLimiter struct {
	mu      sync.Mutex
	entries map[string][]time.Time
	max     int
	window  time.Duration
}

func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		entries: make(map[string][]time.Time),
		max:     max,
		window:  window,
	}
	go rl.cleanupLoop()
	return rl
}

// Allow returns true if the IP is within the allowed rate, false if it should be rejected.
func (rl *RateLimiter) Allow(ip string) bool {
	now := time.Now()
	cutoff := now.Add(-rl.window)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	times := rl.entries[ip]

	// remove timestamps outside the window
	valid := times[:0]
	for _, t := range times {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= rl.max {
		rl.entries[ip] = valid
		return false
	}

	rl.entries[ip] = append(valid, now)
	return true
}

// cleanupLoop periodically evicts IPs whose entire history is outside the window.
// This prevents unbounded map growth when many unique IPs hit the rate-limited endpoint.
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.evictStale()
	}
}

func (rl *RateLimiter) evictStale() {
	cutoff := time.Now().Add(-rl.window)
	rl.mu.Lock()
	defer rl.mu.Unlock()
	for ip, times := range rl.entries {
		valid := times[:0]
		for _, t := range times {
			if t.After(cutoff) {
				valid = append(valid, t)
			}
		}
		if len(valid) == 0 {
			delete(rl.entries, ip)
		} else {
			rl.entries[ip] = valid
		}
	}
}

// LoginRateLimit returns a chi-compatible middleware that limits the route to
// maxAttempts requests per window per client IP. Returns 429 on breach.
func LoginRateLimit(maxAttempts int, window time.Duration) func(http.Handler) http.Handler {
	limiter := NewRateLimiter(maxAttempts, window)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			if !limiter.Allow(ip) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]string{"error": "too many requests, try again later"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// clientIP extracts the real client IP from RemoteAddr only.
// X-Forwarded-For is intentionally ignored: the backend port is only accessible
// from within the Docker network (via nginx), so RemoteAddr is the nginx proxy IP.
// Trusting XFF from an unauthenticated header would allow rate-limit bypass.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
