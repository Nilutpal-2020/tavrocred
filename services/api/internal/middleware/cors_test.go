package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"tavrocred/services/api/internal/middleware"
)

func TestCORSMiddleware(t *testing.T) {
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	t.Run("Preflight OPTIONS with allowed origin", func(t *testing.T) {
		corsMW := middleware.CORS([]string{"http://localhost:3000"})
		h := corsMW(dummyHandler)

		req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/register", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		req.Header.Set("Access-Control-Request-Method", "POST")
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Equal(t, "http://localhost:3000", rec.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(t, rec.Header().Get("Access-Control-Allow-Methods"), "POST")
		assert.Contains(t, rec.Header().Get("Access-Control-Allow-Headers"), "Authorization")
		assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
	})

	t.Run("Preflight OPTIONS automatically maps localhost to 127.0.0.1", func(t *testing.T) {
		corsMW := middleware.CORS([]string{"http://localhost:3000"})
		h := corsMW(dummyHandler)

		req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/register", nil)
		req.Header.Set("Origin", "http://127.0.0.1:3000")
		req.Header.Set("Access-Control-Request-Method", "POST")
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Equal(t, "http://127.0.0.1:3000", rec.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("Preflight OPTIONS with disallowed origin returns 403", func(t *testing.T) {
		corsMW := middleware.CORS([]string{"http://localhost:3000"})
		h := corsMW(dummyHandler)

		req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/register", nil)
		req.Header.Set("Origin", "http://malicious-origin.com")
		req.Header.Set("Access-Control-Request-Method", "POST")
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
		assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("Actual POST request with allowed origin sets headers and forwards request", func(t *testing.T) {
		corsMW := middleware.CORS([]string{"http://localhost:3000"})
		h := corsMW(dummyHandler)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "ok", rec.Body.String())
		assert.Equal(t, "http://localhost:3000", rec.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
	})

	t.Run("Wildcard origin allows any origin", func(t *testing.T) {
		corsMW := middleware.CORS([]string{"*"})
		h := corsMW(dummyHandler)

		req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/register", nil)
		req.Header.Set("Origin", "https://app.example.com")
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Equal(t, "https://app.example.com", rec.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("Empty allowed origins fallback for local development", func(t *testing.T) {
		corsMW := middleware.CORS([]string{})
		h := corsMW(dummyHandler)

		req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/register", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Equal(t, "http://localhost:3000", rec.Header().Get("Access-Control-Allow-Origin"))
	})
}
