package loan_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tavrocred/services/api/internal/middleware"
	"tavrocred/services/api/internal/modules/auth"
	"tavrocred/services/api/internal/modules/loan"
	"tavrocred/services/api/internal/server"
)

func setupLoanTestApp() (http.Handler, *auth.TokenManager, *mockPeerServiceRepo) {
	loanRepo := newMockLoanRepo()
	peerRepo := newMockPeerServiceRepo()
	calc := loan.NewCalculator()
	loanSvc := loan.NewService(loanRepo, peerRepo, calc)
	loanHandler := loan.NewHandler(loanSvc)

	tm := auth.NewTokenManager("test-jwt-secret-for-loan-handlers-32-bytes-long!", 15*time.Minute, 30*24*time.Hour)
	authMiddleware := middleware.Authenticate(tm)

	router := server.NewRouter(server.RouterConfig{
		LoanHandler:    loanHandler,
		AuthMiddleware: authMiddleware,
		RequestIDMW:    middleware.RequestID,
		LoggerMW:       middleware.Logger,
		RecoverMW:      middleware.Recover,
	})

	return router, tm, peerRepo
}

func TestLoanHandler_EndToEndAndOwnershipIsolation(t *testing.T) {
	app, tm, peerRepo := setupLoanTestApp()
	ctx := context.Background()

	userA := uuid.New()
	tokenA, _, _ := tm.GenerateAccessToken(userA, "usera@example.com")

	userB := uuid.New()
	tokenB, _, _ := tm.GenerateAccessToken(userB, "userb@example.com")

	// Create peer for User A
	pA, err := peerRepo.Create(ctx, userA, "Borrower One", nil, nil, nil)
	require.NoError(t, err)

	// 1. User A creates loan in DRAFT status
	createPayload := map[string]any{
		"peer_id":              pA.ID.String(),
		"principal_amount":     "15000.00",
		"currency":             "INR",
		"interest_type":        "SIMPLE",
		"annual_interest_rate": "12.00",
		"start_date":           "2026-01-01",
		"due_date":             "2026-12-31",
		"notes":                "Personal lending to college buddy",
	}
	body, _ := json.Marshal(createPayload)

	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/loans", bytes.NewReader(body))
	reqCreate.Header.Set("Authorization", "Bearer "+tokenA)
	reqCreate.Header.Set("Content-Type", "application/json")
	recCreate := httptest.NewRecorder()

	app.ServeHTTP(recCreate, reqCreate)
	assert.Equal(t, http.StatusCreated, recCreate.Code)

	var createResp struct {
		Success bool              `json:"success"`
		Data    loan.LoanResponse `json:"data"`
	}
	err = json.Unmarshal(recCreate.Body.Bytes(), &createResp)
	require.NoError(t, err)
	loanID := createResp.Data.ID
	assert.Equal(t, "DRAFT", createResp.Data.Status)
	assert.Equal(t, "15000.00", createResp.Data.PrincipalAmount)
	assert.Equal(t, 1, createResp.Data.Version)

	// 2. User A can retrieve loan details
	reqGetA := httptest.NewRequest(http.MethodGet, "/api/v1/loans/"+loanID.String(), nil)
	reqGetA.Header.Set("Authorization", "Bearer "+tokenA)
	recGetA := httptest.NewRecorder()
	app.ServeHTTP(recGetA, reqGetA)
	assert.Equal(t, http.StatusOK, recGetA.Code)

	// 3. User B CANNOT retrieve User A's loan (Strict Owner Isolation -> 404)
	reqGetB := httptest.NewRequest(http.MethodGet, "/api/v1/loans/"+loanID.String(), nil)
	reqGetB.Header.Set("Authorization", "Bearer "+tokenB)
	recGetB := httptest.NewRecorder()
	app.ServeHTTP(recGetB, reqGetB)
	assert.Equal(t, http.StatusNotFound, recGetB.Code)

	// 4. User B CANNOT activate User A's loan
	reqActB := httptest.NewRequest(http.MethodPost, "/api/v1/loans/"+loanID.String()+"/activate", nil)
	reqActB.Header.Set("Authorization", "Bearer "+tokenB)
	recActB := httptest.NewRecorder()
	app.ServeHTTP(recActB, reqActB)
	assert.Equal(t, http.StatusNotFound, recActB.Code)

	// 5. User A activates loan (DRAFT -> ACTIVE)
	reqActA := httptest.NewRequest(http.MethodPost, "/api/v1/loans/"+loanID.String()+"/activate", nil)
	reqActA.Header.Set("Authorization", "Bearer "+tokenA)
	recActA := httptest.NewRecorder()
	app.ServeHTTP(recActA, reqActA)
	assert.Equal(t, http.StatusOK, recActA.Code)

	var actResp struct {
		Success bool              `json:"success"`
		Data    loan.LoanResponse `json:"data"`
	}
	_ = json.Unmarshal(recActA.Body.Bytes(), &actResp)
	assert.Equal(t, "ACTIVE", actResp.Data.Status)

	// 6. User A amends active loan (new version 2)
	amendPayload := map[string]any{
		"principal_amount":     "18000.00",
		"interest_type":        "SIMPLE",
		"annual_interest_rate": "10.00",
		"reason":               "Additional amount lent with reduced rate",
	}
	amendBody, _ := json.Marshal(amendPayload)
	reqAmend := httptest.NewRequest(http.MethodPost, "/api/v1/loans/"+loanID.String()+"/amend", bytes.NewReader(amendBody))
	reqAmend.Header.Set("Authorization", "Bearer "+tokenA)
	reqAmend.Header.Set("Content-Type", "application/json")
	recAmend := httptest.NewRecorder()
	app.ServeHTTP(recAmend, reqAmend)
	assert.Equal(t, http.StatusOK, recAmend.Code)

	var amendResp struct {
		Success bool              `json:"success"`
		Data    loan.LoanResponse `json:"data"`
	}
	_ = json.Unmarshal(recAmend.Body.Bytes(), &amendResp)
	assert.Equal(t, 2, amendResp.Data.Version)
	assert.Equal(t, "18000.00", amendResp.Data.PrincipalAmount)

	// 7. User A views append-only audit events
	reqEvents := httptest.NewRequest(http.MethodGet, "/api/v1/loans/"+loanID.String()+"/events", nil)
	reqEvents.Header.Set("Authorization", "Bearer "+tokenA)
	recEvents := httptest.NewRecorder()
	app.ServeHTTP(recEvents, reqEvents)
	assert.Equal(t, http.StatusOK, recEvents.Code)

	var eventsResp struct {
		Success bool                `json:"success"`
		Data    []*loan.LedgerEvent `json:"data"`
	}
	_ = json.Unmarshal(recEvents.Body.Bytes(), &eventsResp)
	require.Len(t, eventsResp.Data, 3)
	assert.Equal(t, "LOAN_CREATED", eventsResp.Data[0].EventType)
	assert.Equal(t, "LOAN_ACTIVATED", eventsResp.Data[1].EventType)
	assert.Equal(t, "LOAN_AMENDED", eventsResp.Data[2].EventType)
}
