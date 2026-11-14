package middleware

import (
	"net/http"
	"strings"
)

// CORS returns a middleware that handles Cross-Origin Resource Sharing.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	originMap := make(map[string]bool)
	allowAll := false

	for _, o := range allowedOrigins {
		cleaned := strings.TrimRight(strings.TrimSpace(o), "/")
		if cleaned == "*" {
			allowAll = true
		} else if cleaned != "" {
			originMap[cleaned] = true
			// Also support 127.0.0.1 equivalent for localhost
			if strings.HasPrefix(cleaned, "http://localhost:") {
				originMap["http://127.0.0.1:"+strings.TrimPrefix(cleaned, "http://localhost:")] = true
			} else if strings.HasPrefix(cleaned, "http://127.0.0.1:") {
				originMap["http://localhost:"+strings.TrimPrefix(cleaned, "http://127.0.0.1:")] = true
			}
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			cleanedOrigin := strings.TrimRight(strings.TrimSpace(origin), "/")

			isAllowed := false
			if allowAll {
				isAllowed = true
				if origin == "" {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				} else {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				}
			} else if origin != "" && originMap[cleanedOrigin] {
				isAllowed = true
				w.Header().Set("Access-Control-Allow-Origin", origin)
			} else if len(allowedOrigins) == 0 && origin != "" {
				// Default fallback for development
				if cleanedOrigin == "http://localhost:3000" || cleanedOrigin == "http://127.0.0.1:3000" {
					isAllowed = true
					w.Header().Set("Access-Control-Allow-Origin", origin)
				}
			}

			if isAllowed {
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD")
				w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Request-ID, Idempotency-Key, Origin")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID, Content-Length, Content-Type")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}

			// Preflight OPTIONS handling
			if r.Method == http.MethodOptions {
				if isAllowed {
					w.WriteHeader(http.StatusNoContent)
				} else {
					w.WriteHeader(http.StatusForbidden)
				}
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
