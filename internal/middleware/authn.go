package middleware

import (
	"net/http"
	"strings"

	"featureflags/internal/api"
)

// RequireAPIKey returns middleware that requires a valid API key on every
// request. The key may be supplied either as an Authorization header
// ("Authorization: Bearer <key>") or as an X-API-Key header
// ("X-API-Key: <key>"). Requests carrying a missing or mismatching key are
// answered with 401 and the JSON error object {"error":"unauthorized"}.
func RequireAPIKey(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !authorized(key, r) {
				api.WriteError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func authorized(key string, r *http.Request) bool {
	if key == "" {
		return true
	}
	if r.Header.Get("X-API-Key") == key {
		return true
	}
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") && strings.TrimPrefix(auth, "Bearer ") == key {
		return true
	}
	return false
}
