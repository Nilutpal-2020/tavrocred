package borrower_test

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
	"tavrocred/services/api/internal/modules/borrower"
	"tavrocred/services/api/internal/modules/loan"
	"tavrocred/services/api/internal/modules/peer"
	"tavrocred/services/api/internal/server"
)

func setupBorrowerApp() (
	http.Handler,
	*auth.TokenManager,
	borrower.TokenManager,
	*mockBorrowerRepo,
	*mockLoanRepoForBorrowerService,
	uuid.UUID,
	uuid.UUID,
	*loan.Loan,
) {
	bRepo := newMockBorrowerRepo()
	lRepo := newMockLoanRepoForBorrower()
	rRepo := newMockRepaymentRepoForBorrower()
	aRepo := &mockAuthRepoForBorrower{users: make(map[uuid.UUID]*auth.User)}
	calc := loan.NewCalculator()

	jwtSecret := "test-jwt-secret-borrower-handler-32-chars-ok!"
	lenderTM := auth.NewTokenManager(jwtSecret, 15*time.Minute, 30*24*time.Hour)
	borrowerTM := borrower.NewTokenManager(jwtSecret)

	userID := uuid.New()
	peerID := uuid.New()

	aRepo.users[userID] = &auth.User{
		ID:       userID,
		Email:    "lender@example.com",
		FullName: "Lender One",
	}

	testLoan := &loan.Loan{
		ID:                 uuid.New(),
		UserID:             userID,
		PeerID:             peerID,
		PrincipalAmount:    decimal.NewFromInt(15000),
		Currency:           "INR",
		InterestType:       loan.InterestTypeNone,
		AnnualInterestRate: decimal.Zero,
		DayCountConvention: "ACTUAL_365",
		StartDate:          time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Status:             loan.StatusActive,
		Version:            1,
	}
	lRepo.loans[testLoan.ID] = testLoan
	bRepo.loans[testLoan.ID] = testLoan

	bSvc := borrower.NewService(bRepo, lRepo, rRepo, aRepo, calc, borrowerTM, "http://localhost:3000")
	bHandler := borrower.NewHandler(bSvc, borrowerTM)

	lSvc := loan.NewService(lRepo, &mockPeerRepoForBorrowerApp{userID: userID, peerID: peerID}, calc, rRepo)
	lHandler := loan.NewHandler(lSvc)

	authMiddleware := middleware.Authenticate(lenderTM)

	router := server.NewRouter(server.RouterConfig{
		LoanHandler:     lHandler,
		BorrowerHandler: bHandler,
		AuthMiddleware:  authMiddleware,
		RequestIDMW:     middleware.RequestID,
		LoggerMW:        middleware.Logger,
		RecoverMW:       middleware.Recover,
	})

	return router, lenderTM, borrowerTM, bRepo, lRepo, userID, peerID, testLoan
}

type mockPeerRepoForBorrowerApp struct {
	userID uuid.UUID
	peerID uuid.UUID
}

func (m *mockPeerRepoForBorrowerApp) Create(ctx context.Context, userID uuid.UUID, name string, email, phone, notes *string) (*peer.Peer, error) {
	return nil, nil
}

func (m *mockPeerRepoForBorrowerApp) GetByIDAndUser(ctx context.Context, id, userID uuid.UUID) (*peer.Peer, error) {
	return &peer.Peer{
		ID:     m.peerID,
		UserID: m.userID,
		Name:   "Borrower One",
	}, nil
}

func (m *mockPeerRepoForBorrowerApp) ListByUser(ctx context.Context, userID uuid.UUID, search string, limit, offset int32) ([]*peer.Peer, int64, error) {
	return nil, 0, nil
}

func (m *mockPeerRepoForBorrowerApp) Update(ctx context.Context, id, userID uuid.UUID, name string, email, phone, notes *string) (*peer.Peer, error) {
	return nil, nil
}

func (m *mockPeerRepoForBorrowerApp) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return nil
}

func TestBorrowerHandler_FullFlow(t *testing.T) {
	app, lenderTM, _, _, _, userID, _, testLoan := setupBorrowerApp()

	lenderToken, _, err := lenderTM.GenerateAccessToken(userID, "lender@example.com")
	require.NoError(t, err)

	// 1. Lender generates borrower link: POST /api/v1/loans/{loanID}/borrower-link
	linkReqBody, _ := json.Marshal(map[string]any{"expires_in_hours": 72})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/loans/"+testLoan.ID.String()+"/borrower-link", bytes.NewReader(linkReqBody))
	req.Header.Set("Authorization", "Bearer "+lenderToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var linkResp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &linkResp))
	linkData := linkResp["data"].(map[string]any)
	rawToken := linkData["token"].(string)
	assert.NotEmpty(t, rawToken)

	// 2. Borrower exchanges raw token for session: POST /api/v1/public/borrower/access
	accessReqBody, _ := json.Marshal(map[string]any{"token": rawToken})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/public/borrower/access", bytes.NewReader(accessReqBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var accessResp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &accessResp))
	accessData := accessResp["data"].(map[string]any)
	sessionToken := accessData["session_token"].(string)
	assert.NotEmpty(t, sessionToken)

	// 3. Borrower views loan details: GET /api/v1/public/borrower/loan
	req = httptest.NewRequest(http.MethodGet, "/api/v1/public/borrower/loan", nil)
	req.Header.Set("Authorization", "Bearer "+sessionToken)
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var loanResp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &loanResp))
	loanData := loanResp["data"].(map[string]any)
	assert.Equal(t, "15000.00", loanData["loan"].(map[string]any)["principal_amount"])
	assert.Equal(t, "Lender One", loanData["lender"].(map[string]any)["name"])

	// 4. Borrower acknowledges loan: POST /api/v1/public/borrower/acknowledge
	req = httptest.NewRequest(http.MethodPost, "/api/v1/public/borrower/acknowledge", nil)
	req.Header.Set("Authorization", "Bearer "+sessionToken)
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var ackResp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &ackResp))
	assert.Equal(t, "ACKNOWLEDGED", ackResp["data"].(map[string]any)["status"])

	// 5. Borrower disputes loan: POST /api/v1/public/borrower/dispute
	dispBody, _ := json.Marshal(map[string]any{"reason": "Payment date discrepancy"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/public/borrower/dispute", bytes.NewReader(dispBody))
	req.Header.Set("Authorization", "Bearer "+sessionToken)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var dispResp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dispResp))
	assert.Equal(t, "DISPUTED", dispResp["data"].(map[string]any)["status"])

	// 6. Borrower self-reports repayment: POST /api/v1/public/borrower/repayment-reported
	repBody, _ := json.Marshal(map[string]any{
		"amount":         "3500.00",
		"payment_date":   "2026-01-10",
		"payment_method": "UPI",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/public/borrower/repayment-reported", bytes.NewReader(repBody))
	req.Header.Set("Authorization", "Bearer "+sessionToken)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var repResp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &repResp))
	assert.Equal(t, "PENDING_VERIFICATION", repResp["data"].(map[string]any)["status"])
	assert.Equal(t, "BORROWER_REPORTED", repResp["data"].(map[string]any)["source"])

	// 7. Security: Borrower cannot access lender endpoints
	req = httptest.NewRequest(http.MethodGet, "/api/v1/loans", nil)
	req.Header.Set("Authorization", "Bearer "+sessionToken)
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// 8. Security: Unauthenticated request to borrower endpoint is rejected
	req = httptest.NewRequest(http.MethodGet, "/api/v1/public/borrower/loan", nil)
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
