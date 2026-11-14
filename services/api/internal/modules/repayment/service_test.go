package repayment_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internalErrors "tavrocred/services/api/internal/errors"
	"tavrocred/services/api/internal/modules/loan"
	"tavrocred/services/api/internal/modules/repayment"
)

type mockRepaymentRepo struct {
	repayments map[uuid.UUID]*repayment.Repayment
}

func newMockRepaymentRepo() *mockRepaymentRepo {
	return &mockRepaymentRepo{
		repayments: make(map[uuid.UUID]*repayment.Repayment),
	}
}

func (m *mockRepaymentRepo) Create(ctx context.Context, r *repayment.Repayment) (*repayment.Repayment, error) {
	r.ID = uuid.New()
	r.CreatedAt = time.Now().UTC()
	r.UpdatedAt = time.Now().UTC()
	m.repayments[r.ID] = r
	return r, nil
}

func (m *mockRepaymentRepo) GetByIDAndUser(ctx context.Context, id, userID uuid.UUID) (*repayment.Repayment, error) {
	r, exists := m.repayments[id]
	if !exists || r.UserID != userID {
		return nil, internalErrors.NewNotFoundError("repayment not found")
	}
	return r, nil
}

func (m *mockRepaymentRepo) ListByLoan(ctx context.Context, loanID, userID uuid.UUID, status string, limit, offset int32) ([]*repayment.Repayment, int64, error) {
	var filtered []*repayment.Repayment
	for _, r := range m.repayments {
		if r.UserID != userID || r.LoanID != loanID {
			continue
		}
		if status != "" && string(r.Status) != status {
			continue
		}
		filtered = append(filtered, r)
	}
	return filtered, int64(len(filtered)), nil
}

func (m *mockRepaymentRepo) GetConfirmedTotalByLoan(ctx context.Context, loanID, userID uuid.UUID) (decimal.Decimal, error) {
	total := decimal.Zero
	for _, r := range m.repayments {
		if r.UserID == userID && r.LoanID == loanID && r.Status == repayment.StatusConfirmed {
			total = total.Add(r.Amount)
		}
	}
	return total, nil
}

func (m *mockRepaymentRepo) Confirm(ctx context.Context, id, userID, confirmedBy uuid.UUID) (*repayment.Repayment, error) {
	r, exists := m.repayments[id]
	if !exists || r.UserID != userID || r.Status != repayment.StatusPendingVerification {
		return nil, internalErrors.NewNotFoundError("repayment not found or not in pending verification status")
	}
	now := time.Now().UTC()
	r.Status = repayment.StatusConfirmed
	r.ConfirmedAt = &now
	r.ConfirmedBy = &confirmedBy
	r.UpdatedAt = now
	return r, nil
}

func (m *mockRepaymentRepo) Reject(ctx context.Context, id, userID uuid.UUID, reason string) (*repayment.Repayment, error) {
	r, exists := m.repayments[id]
	if !exists || r.UserID != userID || r.Status != repayment.StatusPendingVerification {
		return nil, internalErrors.NewNotFoundError("repayment not found or not in pending verification status")
	}
	now := time.Now().UTC()
	r.Status = repayment.StatusRejected
	r.RejectionReason = &reason
	r.UpdatedAt = now
	return r, nil
}

func (m *mockRepaymentRepo) Reverse(ctx context.Context, id, userID uuid.UUID, reason string) (*repayment.Repayment, error) {
	r, exists := m.repayments[id]
	if !exists || r.UserID != userID || r.Status != repayment.StatusConfirmed {
		return nil, internalErrors.NewNotFoundError("repayment not found or not confirmed")
	}
	now := time.Now().UTC()
	r.Status = repayment.StatusReversed
	r.ReversedAt = &now
	r.ReversalReason = &reason
	r.UpdatedAt = now
	return r, nil
}

type mockLoanRepoForRepayment struct {
	loans  map[uuid.UUID]*loan.Loan
	events map[uuid.UUID][]*loan.LedgerEvent
}

func newMockLoanRepo() *mockLoanRepoForRepayment {
	return &mockLoanRepoForRepayment{
		loans:  make(map[uuid.UUID]*loan.Loan),
		events: make(map[uuid.UUID][]*loan.LedgerEvent),
	}
}

func (m *mockLoanRepoForRepayment) Create(ctx context.Context, l *loan.Loan) (*loan.Loan, error) {
	return nil, nil
}

func (m *mockLoanRepoForRepayment) GetByIDAndUser(ctx context.Context, id, userID uuid.UUID) (*loan.Loan, error) {
	l, exists := m.loans[id]
	if !exists || l.UserID != userID {
		return nil, internalErrors.NewNotFoundError("loan not found")
	}
	return l, nil
}

func (m *mockLoanRepoForRepayment) ListByUser(ctx context.Context, userID uuid.UUID, peerID *uuid.UUID, status string, limit, offset int32) ([]*loan.Loan, int64, error) {
	return nil, 0, nil
}

func (m *mockLoanRepoForRepayment) UpdateDraft(ctx context.Context, l *loan.Loan) (*loan.Loan, error) {
	return nil, nil
}

func (m *mockLoanRepoForRepayment) UpdateStatus(ctx context.Context, id, userID uuid.UUID, newStatus loan.Status) (*loan.Loan, error) {
	l, exists := m.loans[id]
	if !exists || l.UserID != userID {
		return nil, internalErrors.NewNotFoundError("loan not found")
	}
	l.Status = newStatus
	l.UpdatedAt = time.Now().UTC()
	return l, nil
}

func (m *mockLoanRepoForRepayment) Amend(ctx context.Context, id, userID uuid.UUID, principal, rate decimal.Decimal, interestType loan.InterestType, dueDate *time.Time, notes *string) (*loan.Loan, error) {
	return nil, nil
}

func (m *mockLoanRepoForRepayment) CreateLedgerEvent(ctx context.Context, event *loan.LedgerEvent) (*loan.LedgerEvent, error) {
	event.ID = uuid.New()
	event.CreatedAt = time.Now().UTC()
	if event.LoanID != nil {
		m.events[*event.LoanID] = append(m.events[*event.LoanID], event)
	}
	return event, nil
}

func (m *mockLoanRepoForRepayment) ListLedgerEvents(ctx context.Context, loanID, userID uuid.UUID) ([]*loan.LedgerEvent, error) {
	return m.events[loanID], nil
}

func setupRepaymentService() (repayment.Service, *mockRepaymentRepo, *mockLoanRepoForRepayment, uuid.UUID, *loan.Loan) {
	repRepo := newMockRepaymentRepo()
	loanRepo := newMockLoanRepo()
	calc := loan.NewCalculator()
	svc := repayment.NewService(repRepo, loanRepo, calc)

	userID := uuid.New()
	testLoan := &loan.Loan{
		ID:                 uuid.New(),
		UserID:             userID,
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
	loanRepo.loans[testLoan.ID] = testLoan

	return svc, repRepo, loanRepo, userID, testLoan
}

func TestRepaymentService_Record(t *testing.T) {
	ctx := context.Background()

	t.Run("record directly confirmed repayment", func(t *testing.T) {
		svc, _, loanRepo, userID, testLoan := setupRepaymentService()

		req := repayment.RecordRepaymentRequest{
			Amount:            "4000",
			Currency:          "INR",
			PaymentDate:       "2026-01-10",
			PaymentMethod:     "UPI",
			ConfirmedDirectly: true,
		}

		res, err := svc.Record(ctx, testLoan.ID, userID, req)
		require.NoError(t, err)
		assert.Equal(t, "CONFIRMED", res.Status)
		assert.Equal(t, "4000.00", res.Amount)
		assert.NotNil(t, res.ConfirmedAt)
		assert.Equal(t, userID, *res.ConfirmedBy)

		// Check events
		events := loanRepo.events[testLoan.ID]
		require.Len(t, events, 2)
		assert.Equal(t, "REPAYMENT_RECORDED", events[0].EventType)
		assert.Equal(t, "REPAYMENT_CONFIRMED", events[1].EventType)
		assert.Equal(t, loan.StatusActive, testLoan.Status)
	})

	t.Run("record pending verification repayment", func(t *testing.T) {
		svc, _, loanRepo, userID, testLoan := setupRepaymentService()

		req := repayment.RecordRepaymentRequest{
			Amount:            "2000",
			Currency:          "INR",
			PaymentDate:       "2026-01-10",
			PaymentMethod:     "BANK_TRANSFER",
			ConfirmedDirectly: false,
		}

		res, err := svc.Record(ctx, testLoan.ID, userID, req)
		require.NoError(t, err)
		assert.Equal(t, "PENDING_VERIFICATION", res.Status)
		assert.Nil(t, res.ConfirmedAt)

		events := loanRepo.events[testLoan.ID]
		require.Len(t, events, 1)
		assert.Equal(t, "REPAYMENT_RECORDED", events[0].EventType)
	})

	t.Run("overpayment rejected", func(t *testing.T) {
		svc, _, _, userID, testLoan := setupRepaymentService()

		req := repayment.RecordRepaymentRequest{
			Amount:            "12000", // exceeds 10,000 principal
			Currency:          "INR",
			PaymentDate:       "2026-01-10",
			PaymentMethod:     "CASH",
			ConfirmedDirectly: true,
		}

		_, err := svc.Record(ctx, testLoan.ID, userID, req)
		require.Error(t, err)
		var valErr *internalErrors.AppError
		require.ErrorAs(t, err, &valErr)
		assert.Equal(t, "VALIDATION_ERROR", valErr.Code)
		assert.Contains(t, valErr.Details, "amount")
	})

	t.Run("full repayment auto-completes loan", func(t *testing.T) {
		svc, _, loanRepo, userID, testLoan := setupRepaymentService()

		req := repayment.RecordRepaymentRequest{
			Amount:            "10000",
			Currency:          "INR",
			PaymentDate:       "2026-01-15",
			PaymentMethod:     "UPI",
			ConfirmedDirectly: true,
		}

		res, err := svc.Record(ctx, testLoan.ID, userID, req)
		require.NoError(t, err)
		assert.Equal(t, "CONFIRMED", res.Status)

		// Verify loan status updated to COMPLETED
		assert.Equal(t, loan.StatusCompleted, testLoan.Status)

		// Verify LOAN_COMPLETED ledger event created
		events := loanRepo.events[testLoan.ID]
		foundCompleted := false
		for _, e := range events {
			if e.EventType == "LOAN_COMPLETED" {
				foundCompleted = true
				break
			}
		}
		assert.True(t, foundCompleted, "expected LOAN_COMPLETED ledger event")
	})

	t.Run("cannot record repayment on draft loan", func(t *testing.T) {
		svc, _, _, userID, testLoan := setupRepaymentService()
		testLoan.Status = loan.StatusDraft

		req := repayment.RecordRepaymentRequest{
			Amount:            "1000",
			PaymentDate:       "2026-01-10",
			PaymentMethod:     "CASH",
			ConfirmedDirectly: true,
		}

		_, err := svc.Record(ctx, testLoan.ID, userID, req)
		require.Error(t, err)
		var appErr *internalErrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, "RESOURCE_CONFLICT", appErr.Code)
	})
}

func TestRepaymentService_Confirm(t *testing.T) {
	ctx := context.Background()
	svc, repRepo, loanRepo, userID, testLoan := setupRepaymentService()

	// Record a pending repayment
	rep := &repayment.Repayment{
		ID:            uuid.New(),
		UserID:        userID,
		LoanID:        testLoan.ID,
		Amount:        decimal.NewFromInt(10000),
		Currency:      "INR",
		PaymentDate:   time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
		PaymentMethod: "UPI",
		Status:        repayment.StatusPendingVerification,
		Source:        repayment.SourceBorrowerReported,
	}
	repRepo.repayments[rep.ID] = rep

	res, err := svc.Confirm(ctx, rep.ID, userID)
	require.NoError(t, err)
	assert.Equal(t, "CONFIRMED", res.Status)
	assert.NotNil(t, res.ConfirmedAt)
	assert.Equal(t, loan.StatusCompleted, testLoan.Status)

	// Verify ledger event
	events := loanRepo.events[testLoan.ID]
	var foundConfirmed, foundCompleted bool
	for _, e := range events {
		if e.EventType == "REPAYMENT_CONFIRMED" {
			foundConfirmed = true
		}
		if e.EventType == "LOAN_COMPLETED" {
			foundCompleted = true
		}
	}
	assert.True(t, foundConfirmed)
	assert.True(t, foundCompleted)
}

func TestRepaymentService_Reject(t *testing.T) {
	ctx := context.Background()
	svc, repRepo, loanRepo, userID, testLoan := setupRepaymentService()

	rep := &repayment.Repayment{
		ID:            uuid.New(),
		UserID:        userID,
		LoanID:        testLoan.ID,
		Amount:        decimal.NewFromInt(5000),
		Currency:      "INR",
		PaymentDate:   time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
		PaymentMethod: "BANK_TRANSFER",
		Status:        repayment.StatusPendingVerification,
		Source:        repayment.SourceBorrowerReported,
	}
	repRepo.repayments[rep.ID] = rep

	res, err := svc.Reject(ctx, rep.ID, userID, repayment.RejectRepaymentRequest{
		Reason: "Payment not credited to my account",
	})
	require.NoError(t, err)
	assert.Equal(t, "REJECTED", res.Status)
	assert.Equal(t, "Payment not credited to my account", *res.RejectionReason)

	// Verify ledger event
	events := loanRepo.events[testLoan.ID]
	require.Len(t, events, 1)
	assert.Equal(t, "REPAYMENT_REJECTED", events[0].EventType)
}

func TestRepaymentService_Reverse(t *testing.T) {
	ctx := context.Background()
	svc, repRepo, loanRepo, userID, testLoan := setupRepaymentService()

	// Put loan in COMPLETED state
	testLoan.Status = loan.StatusCompleted

	now := time.Now().UTC()
	rep := &repayment.Repayment{
		ID:            uuid.New(),
		UserID:        userID,
		LoanID:        testLoan.ID,
		Amount:        decimal.NewFromInt(10000),
		Currency:      "INR",
		PaymentDate:   time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
		PaymentMethod: "UPI",
		Status:        repayment.StatusConfirmed,
		Source:        repayment.SourceLenderRecorded,
		ConfirmedAt:   &now,
		ConfirmedBy:   &userID,
	}
	repRepo.repayments[rep.ID] = rep

	res, err := svc.Reverse(ctx, rep.ID, userID, repayment.ReverseRepaymentRequest{
		Reason: "Bank bounce notice received",
	})
	require.NoError(t, err)
	assert.Equal(t, "REVERSED", res.Status)
	assert.Equal(t, "Bank bounce notice received", *res.ReversalReason)

	// Loan should be reopened to ACTIVE status!
	assert.Equal(t, loan.StatusActive, testLoan.Status)

	events := loanRepo.events[testLoan.ID]
	var foundReversed, foundReactivated bool
	for _, e := range events {
		if e.EventType == "REPAYMENT_REVERSED" {
			foundReversed = true
		}
		if e.EventType == "LOAN_ACTIVATED" {
			foundReactivated = true
		}
	}
	assert.True(t, foundReversed)
	assert.True(t, foundReactivated)
}

func TestRepaymentService_OwnerScoping(t *testing.T) {
	ctx := context.Background()
	svc, repRepo, _, userID, testLoan := setupRepaymentService()
	otherUserID := uuid.New()

	rep := &repayment.Repayment{
		ID:            uuid.New(),
		UserID:        userID,
		LoanID:        testLoan.ID,
		Amount:        decimal.NewFromInt(1000),
		Currency:      "INR",
		PaymentDate:   time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
		PaymentMethod: "UPI",
		Status:        repayment.StatusConfirmed,
		Source:        repayment.SourceLenderRecorded,
	}
	repRepo.repayments[rep.ID] = rep

	// Other user cannot view this repayment
	_, err := svc.GetByID(ctx, rep.ID, otherUserID)
	require.Error(t, err)
	var notFoundErr *internalErrors.AppError
	require.ErrorAs(t, err, &notFoundErr)
	assert.Equal(t, "NOT_FOUND", notFoundErr.Code)
}
