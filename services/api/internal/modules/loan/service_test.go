package loan_test

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
	"tavrocred/services/api/internal/modules/peer"
)

type mockLoanRepo struct {
	loans  map[uuid.UUID]*loan.Loan
	events map[uuid.UUID][]*loan.LedgerEvent
}

func newMockLoanRepo() *mockLoanRepo {
	return &mockLoanRepo{
		loans:  make(map[uuid.UUID]*loan.Loan),
		events: make(map[uuid.UUID][]*loan.LedgerEvent),
	}
}

func (m *mockLoanRepo) Create(ctx context.Context, l *loan.Loan) (*loan.Loan, error) {
	l.ID = uuid.New()
	l.CreatedAt = time.Now().UTC()
	l.UpdatedAt = time.Now().UTC()
	m.loans[l.ID] = l
	return l, nil
}

func (m *mockLoanRepo) GetByIDAndUser(ctx context.Context, id, userID uuid.UUID) (*loan.Loan, error) {
	l, exists := m.loans[id]
	if !exists || l.UserID != userID {
		return nil, internalErrors.NewNotFoundError("loan not found")
	}
	return l, nil
}

func (m *mockLoanRepo) ListByUser(ctx context.Context, userID uuid.UUID, peerID *uuid.UUID, status string, limit, offset int32) ([]*loan.Loan, int64, error) {
	var filtered []*loan.Loan
	for _, l := range m.loans {
		if l.UserID != userID {
			continue
		}
		if peerID != nil && l.PeerID != *peerID {
			continue
		}
		if status != "" && string(l.Status) != status {
			continue
		}
		filtered = append(filtered, l)
	}

	total := int64(len(filtered))
	start := int(offset)
	if start > len(filtered) {
		return []*loan.Loan{}, total, nil
	}
	end := start + int(limit)
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end], total, nil
}

func (m *mockLoanRepo) UpdateDraft(ctx context.Context, l *loan.Loan) (*loan.Loan, error) {
	existing, exists := m.loans[l.ID]
	if !exists || existing.UserID != l.UserID || existing.Status != loan.StatusDraft {
		return nil, internalErrors.NewNotFoundError("draft loan not found")
	}
	l.UpdatedAt = time.Now().UTC()
	m.loans[l.ID] = l
	return l, nil
}

func (m *mockLoanRepo) UpdateStatus(ctx context.Context, id, userID uuid.UUID, newStatus loan.Status) (*loan.Loan, error) {
	l, exists := m.loans[id]
	if !exists || l.UserID != userID {
		return nil, internalErrors.NewNotFoundError("loan not found")
	}
	l.Status = newStatus
	l.UpdatedAt = time.Now().UTC()
	return l, nil
}

func (m *mockLoanRepo) Amend(ctx context.Context, id, userID uuid.UUID, principal, rate decimal.Decimal, interestType loan.InterestType, dueDate *time.Time, notes *string) (*loan.Loan, error) {
	l, exists := m.loans[id]
	if !exists || l.UserID != userID || l.Status != loan.StatusActive {
		return nil, internalErrors.NewNotFoundError("active loan not found")
	}
	l.PrincipalAmount = principal
	l.AnnualInterestRate = rate
	l.InterestType = interestType
	l.DueDate = dueDate
	l.Notes = notes
	l.Version++
	l.UpdatedAt = time.Now().UTC()
	return l, nil
}

func (m *mockLoanRepo) CreateLedgerEvent(ctx context.Context, event *loan.LedgerEvent) (*loan.LedgerEvent, error) {
	event.ID = uuid.New()
	event.CreatedAt = time.Now().UTC()
	if event.LoanID != nil {
		m.events[*event.LoanID] = append(m.events[*event.LoanID], event)
	}
	return event, nil
}

func (m *mockLoanRepo) ListLedgerEvents(ctx context.Context, loanID, userID uuid.UUID) ([]*loan.LedgerEvent, error) {
	return m.events[loanID], nil
}

type mockPeerServiceRepo struct {
	peers map[uuid.UUID]*peer.Peer
}

func newMockPeerServiceRepo() *mockPeerServiceRepo {
	return &mockPeerServiceRepo{peers: make(map[uuid.UUID]*peer.Peer)}
}

func (m *mockPeerServiceRepo) Create(ctx context.Context, userID uuid.UUID, name string, email, phone, notes *string) (*peer.Peer, error) {
	p := &peer.Peer{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      name,
		Email:     email,
		Phone:     phone,
		Notes:     notes,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	m.peers[p.ID] = p
	return p, nil
}

func (m *mockPeerServiceRepo) GetByIDAndUser(ctx context.Context, id, userID uuid.UUID) (*peer.Peer, error) {
	p, exists := m.peers[id]
	if !exists || p.UserID != userID {
		return nil, internalErrors.NewNotFoundError("peer not found")
	}
	return p, nil
}

func (m *mockPeerServiceRepo) ListByUser(ctx context.Context, userID uuid.UUID, search string, limit, offset int32) ([]*peer.Peer, int64, error) {
	return nil, 0, nil
}
func (m *mockPeerServiceRepo) Update(ctx context.Context, id, userID uuid.UUID, name string, email, phone, notes *string) (*peer.Peer, error) {
	return nil, nil
}
func (m *mockPeerServiceRepo) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return nil
}

func setupLoanService() (loan.Service, *mockLoanRepo, *mockPeerServiceRepo) {
	loanRepo := newMockLoanRepo()
	peerRepo := newMockPeerServiceRepo()
	calc := loan.NewCalculator()
	svc := loan.NewService(loanRepo, peerRepo, calc)
	return svc, loanRepo, peerRepo
}

func TestLoanService_CreateAndGet(t *testing.T) {
	svc, loanRepo, peerRepo := setupLoanService()
	ctx := context.Background()

	userA := uuid.New()
	userB := uuid.New()

	// Create a peer for user A
	pA, err := peerRepo.Create(ctx, userA, "Borrower Alice", nil, nil, nil)
	require.NoError(t, err)

	// User A creates a draft loan
	res, err := svc.Create(ctx, userA, loan.CreateLoanRequest{
		PeerID:             pA.ID,
		PrincipalAmount:    "10000.00",
		Currency:           "INR",
		InterestType:       "SIMPLE",
		AnnualInterestRate: "12.00",
		StartDate:          "2026-01-01",
	})
	require.NoError(t, err)
	assert.Equal(t, "DRAFT", res.Status)
	assert.Equal(t, 1, res.Version)
	assert.Equal(t, "10000.00", res.PrincipalAmount)
	assert.Equal(t, "Borrower Alice", res.PeerName)

	// Verify LOAN_CREATED ledger event was recorded
	events := loanRepo.events[res.ID]
	require.Len(t, events, 1)
	assert.Equal(t, "LOAN_CREATED", events[0].EventType)

	// User B cannot access User A's loan (Owner Scoping)
	_, err = svc.GetByID(ctx, res.ID, userB)
	assert.Error(t, err)
	assert.ErrorIs(t, err, internalErrors.ErrNotFound)

	// User B cannot create a loan targeting User A's peer
	_, err = svc.Create(ctx, userB, loan.CreateLoanRequest{
		PeerID:          pA.ID,
		PrincipalAmount: "5000.00",
		InterestType:    "NONE",
		StartDate:       "2026-01-01",
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, internalErrors.ErrValidation)
}

func TestLoanService_StateTransitions(t *testing.T) {
	svc, loanRepo, peerRepo := setupLoanService()
	ctx := context.Background()
	user := uuid.New()

	p, _ := peerRepo.Create(ctx, user, "Bob", nil, nil, nil)

	created, err := svc.Create(ctx, user, loan.CreateLoanRequest{
		PeerID:          p.ID,
		PrincipalAmount: "20000.00",
		InterestType:    "NONE",
		StartDate:       "2026-01-01",
	})
	require.NoError(t, err)
	assert.Equal(t, "DRAFT", created.Status)

	// 1. Cannot amend while in DRAFT
	_, err = svc.Amend(ctx, created.ID, user, loan.AmendLoanRequest{
		PrincipalAmount: "25000.00",
		InterestType:    "NONE",
		Reason:          "Increase",
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, internalErrors.ErrConflict)

	// 2. Activate loan (DRAFT -> ACTIVE)
	active, err := svc.Activate(ctx, created.ID, user)
	require.NoError(t, err)
	assert.Equal(t, "ACTIVE", active.Status)

	// Verify LOAN_ACTIVATED event recorded
	events := loanRepo.events[created.ID]
	require.Len(t, events, 2)
	assert.Equal(t, "LOAN_ACTIVATED", events[1].EventType)

	// 3. Cannot activate again
	_, err = svc.Activate(ctx, created.ID, user)
	assert.Error(t, err)
	assert.ErrorIs(t, err, internalErrors.ErrConflict)

	// 4. Cannot edit draft once active
	_, err = svc.UpdateDraft(ctx, created.ID, user, loan.UpdateLoanRequest{
		PeerID:          p.ID,
		PrincipalAmount: "15000.00",
		InterestType:    "NONE",
		StartDate:       "2026-01-01",
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, internalErrors.ErrConflict)

	// 5. Amend active loan (increments version)
	amended, err := svc.Amend(ctx, created.ID, user, loan.AmendLoanRequest{
		PrincipalAmount: "25000.00",
		InterestType:    "NONE",
		Reason:          "Mutually agreed increase in principal",
	})
	require.NoError(t, err)
	assert.Equal(t, 2, amended.Version)
	assert.Equal(t, "25000.00", amended.PrincipalAmount)

	// 6. Dispute active loan
	disputed, err := svc.Dispute(ctx, created.ID, user, loan.DisputeLoanRequest{
		Reason: "Borrower disputed principal amount",
	})
	require.NoError(t, err)
	assert.Equal(t, "DISPUTED", disputed.Status)

	// 7. Cancel loan
	cancelled, err := svc.Cancel(ctx, created.ID, user, loan.CancelLoanRequest{
		Reason: "Agreement terminated",
	})
	require.NoError(t, err)
	assert.Equal(t, "CANCELLED", cancelled.Status)
}
