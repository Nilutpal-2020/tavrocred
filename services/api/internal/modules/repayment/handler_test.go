package repayment_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tavrocred/services/api/internal/middleware"
	"tavrocred/services/api/internal/modules/auth"
	"tavrocred/services/api/internal/modules/loan"
	"tavrocred/services/api/internal/modules/peer"
	"tavrocred/services/api/internal/modules/repayment"
	"tavrocred/services/api/internal/server"
)

type mockPeerRepoForRepaymentTest struct {
	peers map[uuid.UUID]*peer.Peer
}

func newMockPeerRepoForRepaymentTest() *mockPeerRepoForRepaymentTest {
	return &mockPeerRepoForRepaymentTest{
		peers: make(map[uuid.UUID]*peer.Peer),
	}
}

func (m *mockPeerRepoForRepaymentTest) Create(ctx context.Context, userID uuid.UUID, name string, email, phone, notes *string) (*peer.Peer, error) {
	p := &peer.Peer{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      name,
		Phone:     phone,
		Email:     email,
		Notes:     notes,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	m.peers[p.ID] = p
	return p, nil
}

func (m *mockPeerRepoForRepaymentTest) GetByIDAndUser(ctx context.Context, id, userID uuid.UUID) (*peer.Peer, error) {
	p, ok := m.peers[id]
	if !ok || p.UserID != userID {
		return nil, assert.AnError
	}
	return p, nil
}

func (m *mockPeerRepoForRepaymentTest) ListByUser(ctx context.Context, userID uuid.UUID, search string, limit, offset int32) ([]*peer.Peer, int64, error) {
	return nil, 0, nil
}

func (m *mockPeerRepoForRepaymentTest) Update(ctx context.Context, id, userID uuid.UUID, name string, email, phone, notes *string) (*peer.Peer, error) {
	p, ok := m.peers[id]
	if !ok || p.UserID != userID {
		return nil, assert.AnError
	}
	p.Name = name
	p.Email = email
	p.Phone = phone
	p.Notes = notes
	p.UpdatedAt = time.Now().UTC()
	return p, nil
}

func (m *mockPeerRepoForRepaymentTest) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return nil
}

func setupRepaymentApp() (http.Handler, *auth.TokenManager, loan.Repository, *mockPeerRepoForRepaymentTest) {
	loanRepo := newMockLoanRepo()
	peerRepo := newMockPeerRepoForRepaymentTest()
	repRepo := newMockRepaymentRepo()
	calc := loan.NewCalculator()

	loanSvc := loan.NewService(loanRepo, peerRepo, calc, repRepo)
	loanHandler := loan.NewHandler(loanSvc)

	repSvc := repayment.NewService(repRepo, loanRepo, calc)
	repHandler := repayment.NewHandler(repSvc)

	tm := auth.NewTokenManager("test-jwt-secret-for-repayment-handlers-32-chars", 15*time.Minute, 30*24*time.Hour)
	authMiddleware := middleware.Authenticate(tm)

	router := server.NewRouter(server.RouterConfig{
		LoanHandler:      loanHandler,
		RepaymentHandler: repHandler,
		AuthMiddleware:   authMiddleware,
		RequestIDMW:      middleware.RequestID,
		LoggerMW:         middleware.Logger,
		RecoverMW:        middleware.Recover,
	})

	return router, tm, loanRepo, peerRepo
}

func TestRepaymentHandler_RoutesAndSecurity(t *testing.T) {
	app, tm, loanRepo, _ := setupRepaymentApp()
	ctx := context.Background()

	userA := uuid.New()
	tokenA, _, _ := tm.GenerateAccessToken(userA, "usera@example.com")

	userB := uuid.New()
	tokenB, _, _ := tm.GenerateAccessToken(userB, "userb@example.com")

	// Create active loan for User A
	testLoan := &loan.Loan{
		ID:                 uuid.New(),
		UserID:             userA,
		PeerID:             uuid.New(),
		PrincipalAmount:    decimal.NewFromInt(10000),
		Currency:           "INR",
		InterestType:       loan.InterestTypeNone,
		AnnualInterestRate: decimal.Zero,
		DayCountConvention: "ACTUAL_365",
		StartDate:          time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Status:             loan.StatusActive,
		Version:            1,
	}
	// Add loan directly to mock
	if m, ok := loanRepo.(*mockLoanRepoForRepayment); ok {
		m.loans[testLoan.ID] = testLoan
	}

	// 1. User A records confirmed repayment
	recPayload := map[string]any{
		"amount":             "3000.00",
		"currency":           "INR",
		"payment_date":       "2026-01-10",
		"payment_method":     "UPI",
		"confirmed_directly": true,
	}
	recBody, _ := json.Marshal(recPayload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/loans/"+testLoan.ID.String()+"/repayments", bytes.NewReader(recBody))
	req.Header.Set("Authorization", "Bearer "+tokenA)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var recResp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &recResp))
	data := recResp["data"].(map[string]any)
	assert.Equal(t, "CONFIRMED", data["status"])
	assert.Equal(t, "3000.00", data["amount"])
	repID := data["id"].(string)

	// 2. User B tries to record repayment on User A's loan -> 404
	req = httptest.NewRequest(http.MethodPost, "/api/v1/loans/"+testLoan.ID.String()+"/repayments", bytes.NewReader(recBody))
	req.Header.Set("Authorization", "Bearer "+tokenB)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// 3. User A lists repayments for the loan
	req = httptest.NewRequest(http.MethodGet, "/api/v1/loans/"+testLoan.ID.String()+"/repayments", nil)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var listResp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &listResp))
	items := listResp["data"].([]any)
	assert.Len(t, items, 1)

	// 4. User B cannot list repayments for User A's loan
	req = httptest.NewRequest(http.MethodGet, "/api/v1/loans/"+testLoan.ID.String()+"/repayments", nil)
	req.Header.Set("Authorization", "Bearer "+tokenB)
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// 5. User A gets repayment by ID
	req = httptest.NewRequest(http.MethodGet, "/api/v1/repayments/"+repID, nil)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// 6. User B cannot get repayment by ID
	req = httptest.NewRequest(http.MethodGet, "/api/v1/repayments/"+repID, nil)
	req.Header.Set("Authorization", "Bearer "+tokenB)
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// 7. User A records a pending repayment
	pendingPayload := map[string]any{
		"amount":             "2000.00",
		"currency":           "INR",
		"payment_date":       "2026-01-12",
		"payment_method":     "BANK_TRANSFER",
		"confirmed_directly": false,
	}
	pendingBody, _ := json.Marshal(pendingPayload)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/loans/"+testLoan.ID.String()+"/repayments", bytes.NewReader(pendingBody))
	req.Header.Set("Authorization", "Bearer "+tokenA)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	var pendResp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &pendResp))
	pendData := pendResp["data"].(map[string]any)
	assert.Equal(t, "PENDING_VERIFICATION", pendData["status"])
	pendingID := pendData["id"].(string)

	// 8. User A rejects the pending repayment
	rejectPayload := map[string]any{"reason": "Did not reflect in account"}
	rejectBody, _ := json.Marshal(rejectPayload)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/repayments/"+pendingID+"/reject", bytes.NewReader(rejectBody))
	req.Header.Set("Authorization", "Bearer "+tokenA)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var rejResp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rejResp))
	rejData := rejResp["data"].(map[string]any)
	assert.Equal(t, "REJECTED", rejData["status"])

	// 9. User A reverses the confirmed repayment
	reversePayload := map[string]any{"reason": "Check bounced"}
	reverseBody, _ := json.Marshal(reversePayload)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/repayments/"+repID+"/reverse", bytes.NewReader(reverseBody))
	req.Header.Set("Authorization", "Bearer "+tokenA)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var revResp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &revResp))
	revData := revResp["data"].(map[string]any)
	assert.Equal(t, "REVERSED", revData["status"])

	_ = ctx
}
