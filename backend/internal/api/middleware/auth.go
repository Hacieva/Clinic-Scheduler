package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Hacieva/clinic-scheduler/backend/internal/auth"
)

type contextKey string

const claimsKey contextKey = "auth_claims"

// Authenticate validates the Bearer JWT and stores claims in the request context.
// Apply only to protected routes; /auth/login must remain public.
func Authenticate(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				writeUnauthorized(w)
				return
			}
			tokenStr := strings.TrimPrefix(header, "Bearer ")
			claims, err := auth.ValidateToken(tokenStr, jwtSecret)
			if err != nil {
				writeUnauthorized(w)
				return
			}
			if claims.TokenType != "access" {
				writeUnauthorized(w)
				return
			}
			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromContext retrieves authenticated user claims stored by Authenticate.
func ClaimsFromContext(ctx context.Context) (*auth.Claims, bool) {
	c, ok := ctx.Value(claimsKey).(*auth.Claims)
	return c, ok
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
}
