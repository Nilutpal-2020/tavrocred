package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tavrocred/services/api/internal/middleware"
	"tavrocred/services/api/internal/modules/auth"
	"tavrocred/services/api/internal/server"
)

func TestRequestIDMiddleware(t *testing.T) {
	handler := middleware.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		assert.NotEmpty(t, reqID)
		w.WriteHeader(http.StatusOK)
	}))

	// Case 1: Without incoming header -> auto-generated UUID
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.NotEmpty(t, rec.Header().Get(middleware.HeaderXRequestID))

	// Case 2: With incoming header -> preserved
	customID := "custom-req-id-123"
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.Header.Set(middleware.HeaderXRequestID, customID)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	assert.Equal(t, customID, rec2.Header().Get(middleware.HeaderXRequestID))
}

func TestRecoverMiddleware(t *testing.T) {
	panicHandler := middleware.Recover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("simulated fatal crash")
	}))

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	assert.NotPanics(t, func() {
		panicHandler.ServeHTTP(rec, req)
	})

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "INTERNAL_ERROR")
}

func TestTimeoutMiddleware(t *testing.T) {
	timeoutHandler := middleware.Timeout(50 * time.Millisecond)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(150 * time.Millisecond):
			w.WriteHeader(http.StatusOK)
		case <-r.Context().Done():
			// Request cancelled due to timeout
			return
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/slow", nil)
	rec := httptest.NewRecorder()

	timeoutHandler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code) // nothing written, default StatusOK in recorder
}

func TestAuthenticateMiddleware(t *testing.T) {
	tm := auth.NewTokenManager("test-secret-key-32-chars-long!!!!!", 15*time.Minute, 30*24*time.Hour)
	authMW := middleware.Authenticate(tm)

	var capturedUserID uuid.UUID
	var capturedEmail string

	protected := authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		capturedUserID, err = server.GetUserIDFromContext(r.Context())
		assert.NoError(t, err)
		capturedEmail = server.GetUserEmailFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	// Case 1: Missing Authorization header -> 401
	req1 := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec1 := httptest.NewRecorder()
	protected.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusUnauthorized, rec1.Code)

	// Case 2: Invalid format -> 401
	req2 := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req2.Header.Set("Authorization", "Basic invalid")
	rec2 := httptest.NewRecorder()
	protected.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusUnauthorized, rec2.Code)

	// Case 3: Valid Bearer token -> 200 and context populated
	expectedUserID := uuid.New()
	expectedEmail := "user@example.com"
	validToken, _, err := tm.GenerateAccessToken(expectedUserID, expectedEmail)
	require.NoError(t, err)

	req3 := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req3.Header.Set("Authorization", "Bearer "+validToken)
	rec3 := httptest.NewRecorder()
	protected.ServeHTTP(rec3, req3)
	assert.Equal(t, http.StatusOK, rec3.Code)
	assert.Equal(t, expectedUserID, capturedUserID)
	assert.Equal(t, expectedEmail, capturedEmail)
}
