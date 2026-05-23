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
	return &RateLimiter{
		entries: make(map[string][]time.Time),
		max:     max,
		window:  window,
	}
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

// clientIP extracts the real client IP, preferring X-Forwarded-For when set.
func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		// X-Forwarded-For can be "client, proxy1, proxy2" — take the first
		if idx := len(fwd); idx > 0 {
			for i := 0; i < len(fwd); i++ {
				if fwd[i] == ',' {
					return fwd[:i]
				}
			}
			return fwd
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
