package server_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"tavrocred/services/api/internal/server"
)

type mockPinger struct {
	err error
}

func (m *mockPinger) Ping(ctx context.Context) error {
	return m.err
}

func TestHealthAndReadinessEndpoints(t *testing.T) {
	// Liveness
	router := server.NewRouter(server.RouterConfig{})
	reqHealth := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recHealth := httptest.NewRecorder()
	router.ServeHTTP(recHealth, reqHealth)
	assert.Equal(t, http.StatusOK, recHealth.Code)
	assert.Contains(t, recHealth.Body.String(), "healthy")

	// Readiness with healthy DB
	healthyRouter := server.NewRouter(server.RouterConfig{
		DB: &mockPinger{err: nil},
	})
	reqReadyHealthy := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	recReadyHealthy := httptest.NewRecorder()
	healthyRouter.ServeHTTP(recReadyHealthy, reqReadyHealthy)
	assert.Equal(t, http.StatusOK, recReadyHealthy.Code)
	assert.Contains(t, recReadyHealthy.Body.String(), "ready")

	// Readiness with failing DB
	failingRouter := server.NewRouter(server.RouterConfig{
		DB: &mockPinger{err: errors.New("db connection refused")},
	})
	reqReadyFailing := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	recReadyFailing := httptest.NewRecorder()
	failingRouter.ServeHTTP(recReadyFailing, reqReadyFailing)
	assert.Equal(t, http.StatusServiceUnavailable, recReadyFailing.Code)
	assert.Contains(t, recReadyFailing.Body.String(), "SERVICE_UNAVAILABLE")
}
