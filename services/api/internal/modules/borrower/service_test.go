package borrower_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internalErrors "tavrocred/services/api/internal/errors"
	"tavrocred/services/api/internal/modules/auth"
	"tavrocred/services/api/internal/modules/borrower"
	"tavrocred/services/api/internal/modules/loan"
	"tavrocred/services/api/internal/modules/repayment"
)

type mockBorrowerRepo struct {
	tokens  map[uuid.UUID]*borrower.BorrowerAccessToken
	acks    map[uuid.UUID][]*borrower.LoanAcknowledgement
	loans   map[uuid.UUID]*loan.Loan
}

func newMockBorrowerRepo() *mockBorrowerRepo {
	return &mockBorrowerRepo{
		tokens: make(map[uuid.UUID]*borrower.BorrowerAccessToken),
		acks:   make(map[uuid.UUID][]*borrower.LoanAcknowledgement),
		loans:  make(map[uuid.UUID]*loan.Loan),
	}
}

func (m *mockBorrowerRepo) CreateAccessToken(ctx context.Context, t *borrower.BorrowerAccessToken) (*borrower.BorrowerAccessToken, error) {
	t.ID = uuid.New()
	t.CreatedAt = time.Now().UTC()
	m.tokens[t.ID] = t
	return t, nil
}

func (m *mockBorrowerRepo) GetActiveAccessTokenByHash(ctx context.Context, tokenHash string) (*borrower.BorrowerAccessToken, error) {
	now := time.Now().UTC()
	for _, t := range m.tokens {
		if t.TokenHash == tokenHash && t.RevokedAt == nil && t.ExpiresAt.After(now) {
			return t, nil
		}
	}
	return nil, internalErrors.NewNotFoundError("valid token not found")
}

func (m *mockBorrowerRepo) RevokeAccessTokensByLoan(ctx context.Context, loanID, userID uuid.UUID) error {
	now := time.Now().UTC()
	for _, t := range m.tokens {
		if t.LoanID == loanID && t.UserID == userID && t.RevokedAt == nil {
			t.RevokedAt = &now
		}
	}
	return nil
}

func (m *mockBorrowerRepo) MarkAccessTokenUsed(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	if t, ok := m.tokens[id]; ok {
		t.UsedAt = &now
	}
	return nil
}

func (m *mockBorrowerRepo) CreateAcknowledgement(ctx context.Context, ack *borrower.LoanAcknowledgement) (*borrower.LoanAcknowledgement, error) {
	ack.ID = uuid.New()
	ack.CreatedAt = time.Now().UTC()
	m.acks[ack.LoanID] = append(m.acks[ack.LoanID], ack)
	return ack, nil
}

func (m *mockBorrowerRepo) GetLatestAcknowledgementByLoan(ctx context.Context, loanID uuid.UUID) (*borrower.LoanAcknowledgement, error) {
	acks := m.acks[loanID]
	if len(acks) == 0 {
		return nil, internalErrors.NewNotFoundError("no acknowledgement found")
	}
	return acks[len(acks)-1], nil
}

func (m *mockBorrowerRepo) ListAcknowledgementsByLoan(ctx context.Context, loanID uuid.UUID) ([]*borrower.LoanAcknowledgement, error) {
	return m.acks[loanID], nil
}

func (m *mockBorrowerRepo) GetLoanByIDAndPeer(ctx context.Context, loanID, peerID uuid.UUID) (*loan.Loan, error) {
	l, ok := m.loans[loanID]
	if !ok || l.PeerID != peerID {
		return nil, internalErrors.NewNotFoundError("loan not found for this peer")
	}
	return l, nil
}

func (m *mockBorrowerRepo) UpdateLoanStatusByPeer(ctx context.Context, loanID, peerID uuid.UUID, status loan.Status) (*loan.Loan, error) {
	l, ok := m.loans[loanID]
	if !ok || l.PeerID != peerID {
		return nil, internalErrors.NewNotFoundError("loan not found for this peer")
	}
	l.Status = status
	l.UpdatedAt = time.Now().UTC()
	return l, nil
}

type mockLoanRepoForBorrowerService struct {
	loans  map[uuid.UUID]*loan.Loan
	events map[uuid.UUID][]*loan.LedgerEvent
}

func newMockLoanRepoForBorrower() *mockLoanRepoForBorrowerService {
	return &mockLoanRepoForBorrowerService{
		loans:  make(map[uuid.UUID]*loan.Loan),
		events: make(map[uuid.UUID][]*loan.LedgerEvent),
	}
}

func (m *mockLoanRepoForBorrowerService) Create(ctx context.Context, l *loan.Loan) (*loan.Loan, error) {
	return nil, nil
}

func (m *mockLoanRepoForBorrowerService) GetByIDAndUser(ctx context.Context, id, userID uuid.UUID) (*loan.Loan, error) {
	l, ok := m.loans[id]
	if !ok || l.UserID != userID {
		return nil, internalErrors.NewNotFoundError("loan not found")
	}
	return l, nil
}

func (m *mockLoanRepoForBorrowerService) ListByUser(ctx context.Context, userID uuid.UUID, peerID *uuid.UUID, status string, limit, offset int32) ([]*loan.Loan, int64, error) {
	return nil, 0, nil
}

func (m *mockLoanRepoForBorrowerService) UpdateDraft(ctx context.Context, l *loan.Loan) (*loan.Loan, error) {
	return nil, nil
}

func (m *mockLoanRepoForBorrowerService) UpdateStatus(ctx context.Context, id, userID uuid.UUID, newStatus loan.Status) (*loan.Loan, error) {
	return nil, nil
}

func (m *mockLoanRepoForBorrowerService) Amend(ctx context.Context, id, userID uuid.UUID, principal, rate decimal.Decimal, interestType loan.InterestType, dueDate *time.Time, notes *string) (*loan.Loan, error) {
	return nil, nil
}

func (m *mockLoanRepoForBorrowerService) CreateLedgerEvent(ctx context.Context, event *loan.LedgerEvent) (*loan.LedgerEvent, error) {
	event.ID = uuid.New()
	event.CreatedAt = time.Now().UTC()
	if event.LoanID != nil {
		m.events[*event.LoanID] = append(m.events[*event.LoanID], event)
	}
	return event, nil
}

func (m *mockLoanRepoForBorrowerService) ListLedgerEvents(ctx context.Context, loanID, userID uuid.UUID) ([]*loan.LedgerEvent, error) {
	return m.events[loanID], nil
}

type mockRepaymentRepoForBorrower struct {
	repayments map[uuid.UUID]*repayment.Repayment
}

func newMockRepaymentRepoForBorrower() *mockRepaymentRepoForBorrower {
	return &mockRepaymentRepoForBorrower{
		repayments: make(map[uuid.UUID]*repayment.Repayment),
	}
}

func (m *mockRepaymentRepoForBorrower) Create(ctx context.Context, r *repayment.Repayment) (*repayment.Repayment, error) {
	r.ID = uuid.New()
	r.CreatedAt = time.Now().UTC()
	r.UpdatedAt = time.Now().UTC()
	m.repayments[r.ID] = r
	return r, nil
}

func (m *mockRepaymentRepoForBorrower) GetByIDAndUser(ctx context.Context, id, userID uuid.UUID) (*repayment.Repayment, error) {
	return nil, nil
}

func (m *mockRepaymentRepoForBorrower) ListByLoan(ctx context.Context, loanID, userID uuid.UUID, status string, limit, offset int32) ([]*repayment.Repayment, int64, error) {
	var list []*repayment.Repayment
	for _, r := range m.repayments {
		if r.LoanID == loanID {
			list = append(list, r)
		}
	}
	return list, int64(len(list)), nil
}

func (m *mockRepaymentRepoForBorrower) GetConfirmedTotalByLoan(ctx context.Context, loanID, userID uuid.UUID) (decimal.Decimal, error) {
	total := decimal.Zero
	for _, r := range m.repayments {
		if r.LoanID == loanID && r.Status == repayment.StatusConfirmed {
			total = total.Add(r.Amount)
		}
	}
	return total, nil
}

func (m *mockRepaymentRepoForBorrower) Confirm(ctx context.Context, id, userID, confirmedBy uuid.UUID) (*repayment.Repayment, error) {
	return nil, nil
}

func (m *mockRepaymentRepoForBorrower) Reject(ctx context.Context, id, userID uuid.UUID, reason string) (*repayment.Repayment, error) {
	return nil, nil
}

func (m *mockRepaymentRepoForBorrower) Reverse(ctx context.Context, id, userID uuid.UUID, reason string) (*repayment.Repayment, error) {
	return nil, nil
}

type mockAuthRepoForBorrower struct {
	users map[uuid.UUID]*auth.User
}

func (m *mockAuthRepoForBorrower) CreateUser(ctx context.Context, email, passwordHash, fullName string) (*auth.User, error) {
	return nil, nil
}

func (m *mockAuthRepoForBorrower) GetUserByID(ctx context.Context, id uuid.UUID) (*auth.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, internalErrors.NewNotFoundError("user not found")
	}
	return u, nil
}

func (m *mockAuthRepoForBorrower) GetUserByEmail(ctx context.Context, email string) (*auth.User, error) {
	return nil, nil
}

func (m *mockAuthRepoForBorrower) CreateRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (*auth.RefreshToken, error) {
	return nil, nil
}

func (m *mockAuthRepoForBorrower) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*auth.RefreshToken, error) {
	return nil, nil
}

func (m *mockAuthRepoForBorrower) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	return nil
}

func (m *mockAuthRepoForBorrower) RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	return nil
}

func setupBorrowerServiceTest() (
	borrower.Service,
	*mockBorrowerRepo,
	*mockLoanRepoForBorrowerService,
	*mockRepaymentRepoForBorrower,
	uuid.UUID,
	uuid.UUID,
	*loan.Loan,
) {
	bRepo := newMockBorrowerRepo()
	lRepo := newMockLoanRepoForBorrower()
	rRepo := newMockRepaymentRepoForBorrower()
	aRepo := &mockAuthRepoForBorrower{users: make(map[uuid.UUID]*auth.User)}
	calc := loan.NewCalculator()
	tm := borrower.NewTokenManager("test-jwt-secret-borrower-service-32-chars!")

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
		PrincipalAmount:    decimal.NewFromInt(20000),
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

	svc := borrower.NewService(bRepo, lRepo, rRepo, aRepo, calc, tm, "http://localhost:3000")
	return svc, bRepo, lRepo, rRepo, userID, peerID, testLoan
}

func TestBorrowerService_GenerateAndExchangeLink(t *testing.T) {
	ctx := context.Background()
	svc, bRepo, _, _, userID, peerID, testLoan := setupBorrowerServiceTest()

	t.Run("generate borrower link", func(t *testing.T) {
		res, err := svc.GenerateBorrowerLink(ctx, testLoan.ID, userID, 48*time.Hour)
		require.NoError(t, err)
		assert.NotEmpty(t, res.Token)
		assert.Contains(t, res.URL, "/portal/loans?token="+res.Token)

		// Verify token stored with SHA-256 hash in DB
		found := false
		for _, tok := range bRepo.tokens {
			if tok.LoanID == testLoan.ID && tok.PeerID == peerID {
				found = true
				assert.NotEqual(t, res.Token, tok.TokenHash) // hash != plaintext
				assert.Equal(t, borrower.PurposePortalAccess, tok.Purpose)
			}
		}
		assert.True(t, found)

		// Exchange token for session
		accessResp, err := svc.ExchangeTokenForSession(ctx, res.Token)
		require.NoError(t, err)
		assert.Equal(t, testLoan.ID, accessResp.LoanID)
		assert.NotEmpty(t, accessResp.SessionToken)
	})

	t.Run("cannot exchange invalid token", func(t *testing.T) {
		_, err := svc.ExchangeTokenForSession(ctx, "invalid-random-token-does-not-exist-in-db-123456789")
		require.Error(t, err)
	})

	t.Run("revoked token cannot be exchanged", func(t *testing.T) {
		res, err := svc.GenerateBorrowerLink(ctx, testLoan.ID, userID, 48*time.Hour)
		require.NoError(t, err)

		err = svc.RevokeBorrowerLink(ctx, testLoan.ID, userID)
		require.NoError(t, err)

		_, err = svc.ExchangeTokenForSession(ctx, res.Token)
		require.Error(t, err)
	})
}

func TestBorrowerService_GetLoanDetails(t *testing.T) {
	ctx := context.Background()
	svc, _, _, rRepo, _, peerID, testLoan := setupBorrowerServiceTest()

	// Add a confirmed repayment
	rRepo.repayments[uuid.New()] = &repayment.Repayment{
		ID:            uuid.New(),
		UserID:        testLoan.UserID,
		LoanID:        testLoan.ID,
		Amount:        decimal.NewFromInt(5000),
		Currency:      "INR",
		Status:        repayment.StatusConfirmed,
		PaymentMethod: "UPI",
		PaymentDate:   time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
	}

	res, err := svc.GetLoanDetailsForBorrower(ctx, testLoan.ID, peerID)
	require.NoError(t, err)
	assert.Equal(t, "20000.00", res.Loan.PrincipalAmount)
	assert.Equal(t, "Lender One", res.Lender.Name)
	assert.Equal(t, "lender@example.com", res.Lender.Email)
	assert.Equal(t, "15000.00", res.Summary.OutstandingBalance.StringFixed(2))
	assert.Len(t, res.Repayments, 1)
}

func TestBorrowerService_AcknowledgeAndDispute(t *testing.T) {
	ctx := context.Background()
	svc, _, lRepo, _, _, peerID, testLoan := setupBorrowerServiceTest()

	t.Run("acknowledge loan terms", func(t *testing.T) {
		ack, err := svc.AcknowledgeLoan(ctx, testLoan.ID, peerID, "127.0.0.1", "Mozilla/5.0")
		require.NoError(t, err)
		assert.Equal(t, borrower.StatusAcknowledged, ack.Status)
		assert.Equal(t, testLoan.Version, ack.Version)

		events := lRepo.events[testLoan.ID]
		foundAck := false
		for _, e := range events {
			if e.EventType == "BORROWER_ACKNOWLEDGED" {
				foundAck = true
				break
			}
		}
		assert.True(t, foundAck)
	})

	t.Run("dispute loan terms", func(t *testing.T) {
		ack, err := svc.DisputeLoan(ctx, testLoan.ID, peerID, "Dispute: Start date is wrong", "127.0.0.1", "Mozilla/5.0")
		require.NoError(t, err)
		assert.Equal(t, borrower.StatusDisputed, ack.Status)
		assert.Equal(t, "Dispute: Start date is wrong", *ack.DisputeReason)
		assert.Equal(t, loan.StatusDisputed, testLoan.Status)

		events := lRepo.events[testLoan.ID]
		foundDisp := false
		for _, e := range events {
			if e.EventType == "LOAN_DISPUTED" {
				foundDisp = true
				break
			}
		}
		assert.True(t, foundDisp)
	})

	t.Run("acknowledgement clears dispute back to active", func(t *testing.T) {
		testLoan.Status = loan.StatusDisputed
		ack, err := svc.AcknowledgeLoan(ctx, testLoan.ID, peerID, "127.0.0.1", "Mozilla/5.0")
		require.NoError(t, err)
		assert.Equal(t, borrower.StatusAcknowledged, ack.Status)
		assert.Equal(t, loan.StatusActive, testLoan.Status)
	})
}

func TestBorrowerService_ReportRepayment(t *testing.T) {
	ctx := context.Background()
	svc, _, lRepo, _, _, peerID, testLoan := setupBorrowerServiceTest()

	t.Run("report valid repayment", func(t *testing.T) {
		req := borrower.BorrowerReportRepaymentRequest{
			Amount:            "4000.00",
			PaymentDate:       "2026-01-15",
			PaymentMethod:     "UPI",
			ExternalReference: strPtr("UPI-REF-001"),
			Notes:             strPtr("Transferred to your GPay"),
		}

		res, err := svc.ReportRepayment(ctx, testLoan.ID, peerID, req)
		require.NoError(t, err)
		assert.Equal(t, "PENDING_VERIFICATION", res.Status)
		assert.Equal(t, "BORROWER_REPORTED", res.Source)
		assert.Equal(t, "4000.00", res.Amount)

		// Audit log
		events := lRepo.events[testLoan.ID]
		foundRep := false
		for _, e := range events {
			if e.EventType == "REPAYMENT_REPORTED" {
				foundRep = true
				break
			}
		}
		assert.True(t, foundRep)
	})

	t.Run("cannot overpay balance", func(t *testing.T) {
		req := borrower.BorrowerReportRepaymentRequest{
			Amount:        "50000.00", // exceeds 20,000
			PaymentDate:   "2026-01-15",
			PaymentMethod: "UPI",
		}

		_, err := svc.ReportRepayment(ctx, testLoan.ID, peerID, req)
		require.Error(t, err)
		var valErr *internalErrors.AppError
		require.ErrorAs(t, err, &valErr)
		assert.Equal(t, "VALIDATION_ERROR", valErr.Code)
		assert.Contains(t, valErr.Details, "amount")
	})
}
