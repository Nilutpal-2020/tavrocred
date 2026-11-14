package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type contextKey string

const (
	RequestIDKey     contextKey = "request_id"
	HeaderXRequestID            = "X-Request-ID"
)

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get(HeaderXRequestID)
		if reqID == "" {
			reqID = uuid.NewString()
		}

		ctx := context.WithValue(r.Context(), RequestIDKey, reqID)
		w.Header().Set(HeaderXRequestID, reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetRequestID(ctx context.Context) string {
	if reqID, ok := ctx.Value(RequestIDKey).(string); ok {
		return reqID
	}
	return ""
}
