package auth_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tavrocred/services/api/internal/middleware"
	"tavrocred/services/api/internal/modules/auth"
	"tavrocred/services/api/internal/server"
)

func setupTestApp() (http.Handler, *auth.TokenManager) {
	repo := newMockAuthRepo()
	hasher := &auth.PasswordHasher{
		Memory:      16 * 1024,
		Iterations:  2,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	}
	tm := auth.NewTokenManager("test-secret-for-http-handlers-at-least-32-chars!", 15*time.Minute, 30*24*time.Hour)
	authSvc := auth.NewService(repo, hasher, tm)
	authHandler := auth.NewHandler(authSvc)

	authMiddleware := middleware.Authenticate(tm)

	router := server.NewRouter(server.RouterConfig{
		AuthHandler:    authHandler,
		AuthMiddleware: authMiddleware,
		RequestIDMW:    middleware.RequestID,
		LoggerMW:       middleware.Logger,
		RecoverMW:      middleware.Recover,
	})

	return router, tm
}

func TestAuthHandler_RegisterAndLoginFlow(t *testing.T) {
	app, _ := setupTestApp()

	// 1. Register valid user
	regPayload := map[string]string{
		"email":     "alice@example.com",
		"password":  "Secret123!",
		"full_name": "Alice Allison",
	}
	body, _ := json.Marshal(regPayload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var regResp struct {
		Success bool `json:"success"`
		Data    struct {
			AccessToken  string            `json:"access_token"`
			RefreshToken string            `json:"refresh_token"`
			User         auth.UserResponse `json:"user"`
		} `json:"data"`
	}
	err := json.Unmarshal(rec.Body.Bytes(), &regResp)
	require.NoError(t, err)
	assert.True(t, regResp.Success)
	assert.NotEmpty(t, regResp.Data.AccessToken)
	assert.NotEmpty(t, regResp.Data.RefreshToken)
	assert.Equal(t, "alice@example.com", regResp.Data.User.Email)

	// 2. Duplicate registration returns 409 Conflict
	recConflict := httptest.NewRecorder()
	reqConflict := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	reqConflict.Header.Set("Content-Type", "application/json")
	app.ServeHTTP(recConflict, reqConflict)
	assert.Equal(t, http.StatusConflict, recConflict.Code)

	// 3. Login with correct credentials returns 200 OK
	loginPayload := map[string]string{
		"email":    "alice@example.com",
		"password": "Secret123!",
	}
	loginBody, _ := json.Marshal(loginPayload)
	reqLogin := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	reqLogin.Header.Set("Content-Type", "application/json")
	recLogin := httptest.NewRecorder()

	app.ServeHTTP(recLogin, reqLogin)
	assert.Equal(t, http.StatusOK, recLogin.Code)

	// 4. Access protected /api/v1/auth/me with Bearer token
	reqMe := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+regResp.Data.AccessToken)
	recMe := httptest.NewRecorder()

	app.ServeHTTP(recMe, reqMe)
	assert.Equal(t, http.StatusOK, recMe.Code)

	// 5. Access protected /api/v1/auth/me without token returns 401 Unauthorized
	reqMeNoAuth := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	recMeNoAuth := httptest.NewRecorder()

	app.ServeHTTP(recMeNoAuth, reqMeNoAuth)
	assert.Equal(t, http.StatusUnauthorized, recMeNoAuth.Code)

	// 6. Refresh token rotation
	refreshPayload := map[string]string{
		"refresh_token": regResp.Data.RefreshToken,
	}
	refreshBody, _ := json.Marshal(refreshPayload)
	reqRefresh := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(refreshBody))
	reqRefresh.Header.Set("Content-Type", "application/json")
	recRefresh := httptest.NewRecorder()

	app.ServeHTTP(recRefresh, reqRefresh)
	assert.Equal(t, http.StatusOK, recRefresh.Code)

	var refResp struct {
		Success bool `json:"success"`
		Data    struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	err = json.Unmarshal(recRefresh.Body.Bytes(), &refResp)
	require.NoError(t, err)
	assert.NotEqual(t, regResp.Data.RefreshToken, refResp.Data.RefreshToken)

	// 7. Logout
	logoutPayload := map[string]string{
		"refresh_token": refResp.Data.RefreshToken,
	}
	logoutBody, _ := json.Marshal(logoutPayload)
	reqLogout := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewReader(logoutBody))
	reqLogout.Header.Set("Content-Type", "application/json")
	recLogout := httptest.NewRecorder()

	app.ServeHTTP(recLogout, reqLogout)
	assert.Equal(t, http.StatusOK, recLogout.Code)
}
