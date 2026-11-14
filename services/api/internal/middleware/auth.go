package middleware

import (
	"net/http"
	"strings"

	"tavrocred/services/api/internal/modules/auth"
	"tavrocred/services/api/internal/server"
)

func Authenticate(tokenManager *auth.TokenManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				server.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authorization header", nil)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				server.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid authorization header format, expected 'Bearer <token>'", nil)
				return
			}

			tokenStr := strings.TrimSpace(parts[1])
			claims, err := tokenManager.ValidateAccessToken(tokenStr)
			if err != nil {
				server.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired access token", nil)
				return
			}

			ctx := server.SetUserContext(r.Context(), claims.UserID, claims.Email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
