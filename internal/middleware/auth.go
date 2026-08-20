package middleware

import (
	"crypto/subtle"
	"net/http"
)

// RequireAPIKey rejects any request that doesn't present the configured key
// via the X-API-Key header. It's a pre-shared secret, not a full auth
// system -- appropriate for a single trusted-caller API sitting behind the
// NetworkPolicy described in docs/SECURITY.md; upgrade to JWT/mTLS if this
// ever grows multiple distinct callers that need per-caller identity.
func RequireAPIKey(expected string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got := r.Header.Get("X-API-Key")
			if expected == "" || subtle.ConstantTimeCompare([]byte(got), []byte(expected)) != 1 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"missing or invalid API key"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
