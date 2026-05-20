package middleware

import "net/http"

// BotAuth validates the X-Bot-Token header against the expected bot secret.
// Stateless: no JWT, no context mutation.
// If secret is empty (misconfigured server) all requests are rejected.
func BotAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if secret == "" || r.Header.Get("X-Bot-Token") != secret {
				writeUnauthorized(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
