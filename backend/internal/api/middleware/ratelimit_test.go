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

func TestLoginRateLimit_XForwardedFor(t *testing.T) {
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

	assert.Equal(t, http.StatusOK, makeReq("192.168.1.1"))
	assert.Equal(t, http.StatusOK, makeReq("192.168.1.1"))
	assert.Equal(t, http.StatusTooManyRequests, makeReq("192.168.1.1"))
	assert.Equal(t, http.StatusOK, makeReq("192.168.1.2"), "different IP must not be blocked")
}
