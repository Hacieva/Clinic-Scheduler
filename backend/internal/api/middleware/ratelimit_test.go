package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimiter_AllowsUnderLimit(t *testing.T) {
	rl := NewRateLimiter(5, time.Minute)
	for range 5 {
		assert.True(t, rl.Allow("1.2.3.4"))
	}
}

func TestRateLimiter_BlocksOverLimit(t *testing.T) {
	rl := NewRateLimiter(5, time.Minute)
	for range 5 {
		rl.Allow("1.2.3.4")
	}
	assert.False(t, rl.Allow("1.2.3.4"))
}

func TestRateLimiter_IndependentPerIP(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)
	rl.Allow("1.1.1.1")
	rl.Allow("1.1.1.1")
	assert.False(t, rl.Allow("1.1.1.1"))
	assert.True(t, rl.Allow("2.2.2.2"))
}

func TestRateLimiter_WindowExpiry(t *testing.T) {
	rl := NewRateLimiter(2, 50*time.Millisecond)
	rl.Allow("1.2.3.4")
	rl.Allow("1.2.3.4")
	assert.False(t, rl.Allow("1.2.3.4"))

	time.Sleep(60 * time.Millisecond)
	assert.True(t, rl.Allow("1.2.3.4"), "window should have expired")
}

func TestLoginRateLimit_Returns429AfterLimit(t *testing.T) {
	handler := LoginRateLimit(3, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	makeReq := func() int {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	require.Equal(t, http.StatusOK, makeReq())
	require.Equal(t, http.StatusOK, makeReq())
	require.Equal(t, http.StatusOK, makeReq())
	assert.Equal(t, http.StatusTooManyRequests, makeReq())
}

// TestLoginRateLimit_XForwardedForIgnored verifies that X-Forwarded-For is NOT trusted.
// All requests sharing the same RemoteAddr must be rate-limited together, regardless of
// what XFF header the client supplies. Trusting a client-supplied XFF would allow
// trivial bypass of the rate limit.
func TestLoginRateLimit_XForwardedForIgnored(t *testing.T) {
	handler := LoginRateLimit(2, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	makeReq := func(xff string) int {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
		req.RemoteAddr = "proxy:9999"
		req.Header.Set("X-Forwarded-For", xff)
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	// Two requests succeed (limit=2), both from the same RemoteAddr.
	assert.Equal(t, http.StatusOK, makeReq("192.168.1.1"))
	assert.Equal(t, http.StatusOK, makeReq("192.168.1.1"))
	// Third request from the same RemoteAddr is blocked even with a different XFF value —
	// XFF is ignored so rotating it must not bypass the limit.
	assert.Equal(t, http.StatusTooManyRequests, makeReq("192.168.1.2"), "spoofed XFF must not bypass rate limit")
}
