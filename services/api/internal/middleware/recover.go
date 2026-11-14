package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"tavrocred/services/api/internal/server"
)

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				if rvr == http.ErrAbortHandler {
					panic(rvr)
				}

				stack := debug.Stack()
				reqID := GetRequestID(r.Context())

				slog.Error("panic recovered",
					"error", fmt.Sprintf("%v", rvr),
					"request_id", reqID,
					"stack", string(stack),
				)

				server.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred", nil)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
