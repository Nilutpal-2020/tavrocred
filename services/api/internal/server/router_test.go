package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"tavrocred/services/api/internal/middleware"
	"tavrocred/services/api/internal/server"
)

type dummyAuthHandler struct{}

func (d *dummyAuthHandler) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/auth", func(auth chi.Router) {
		auth.Post("/register", func(w http.ResponseWriter, r *http.Request) {
			server.JSON(w, http.StatusCreated, map[string]string{"message": "registered"})
		})
	})
}

func TestRouter_CORS(t *testing.T) {
	corsMW := middleware.CORS([]string{"http://localhost:3000"})
	router := server.NewRouter(server.RouterConfig{
		CorsMW:      corsMW,
		AuthHandler: &dummyAuthHandler{},
	})

	t.Run("OPTIONS /api/v1/auth/register preflight", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/register", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		req.Header.Set("Access-Control-Request-Method", "POST")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Equal(t, "http://localhost:3000", rec.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(t, rec.Header().Get("Access-Control-Allow-Methods"), "POST")
		assert.Contains(t, rec.Header().Get("Access-Control-Allow-Headers"), "Authorization")
		assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
	})

	t.Run("POST /api/v1/auth/register with Origin header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Equal(t, "http://localhost:3000", rec.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
	})
}
