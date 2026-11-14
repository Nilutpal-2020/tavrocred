package loan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	internalErrors "tavrocred/services/api/internal/errors"
	"tavrocred/services/api/internal/modules/peer"
	"tavrocred/services/api/internal/server"
)

type Service interface {
	Create(ctx context.Context, userID uuid.UUID, req CreateLoanRequest) (*LoanResponse, error)
	GetByID(ctx context.Context, id, userID uuid.UUID) (*LoanResponse, error)
	List(ctx context.Context, userID uuid.UUID, query ListLoansQuery) ([]LoanResponse, *server.Meta, error)
	UpdateDraft(ctx context.Context, id, userID uuid.UUID, req UpdateLoanRequest) (*LoanResponse, error)
	Activate(ctx context.Context, id, userID uuid.UUID) (*LoanResponse, error)
	Cancel(ctx context.Context, id, userID uuid.UUID, req CancelLoanRequest) (*LoanResponse, error)
	Dispute(ctx context.Context, id, userID uuid.UUID, req DisputeLoanRequest) (*LoanResponse, error)
	Amend(ctx context.Context, id, userID uuid.UUID, req AmendLoanRequest) (*LoanResponse, error)
	ListAuditEvents(ctx context.Context, id, userID uuid.UUID) ([]*LedgerEvent, error)
}

type RepaymentSummaryProvider interface {
	GetConfirmedTotalByLoan(ctx context.Context, loanID, userID uuid.UUID) (decimal.Decimal, error)
}

type loanService struct {
	repo              Repository
	peerRepo          peer.Repository
	calculator        *Calculator
	repaymentProvider RepaymentSummaryProvider
}

func NewService(repo Repository, peerRepo peer.Repository, calculator *Calculator, repaymentProvider ...RepaymentSummaryProvider) Service {
	var rp RepaymentSummaryProvider
	if len(repaymentProvider) > 0 {
		rp = repaymentProvider[0]
	}
	return &loanService{
		repo:              repo,
		peerRepo:          peerRepo,
		calculator:        calculator,
		repaymentProvider: rp,
	}
}

func (s *loanService) getConfirmedRepayments(ctx context.Context, loanID, userID uuid.UUID) decimal.Decimal {
	if s.repaymentProvider != nil {
		total, err := s.repaymentProvider.GetConfirmedTotalByLoan(ctx, loanID, userID)
		if err == nil {
			return total
		}
	}
	return decimal.Zero
}

func (s *loanService) Create(ctx context.Context, userID uuid.UUID, req CreateLoanRequest) (*LoanResponse, error) {
	principal, rate, startDate, dueDate, err := req.SanitizeAndValidate()
	if err != nil {
		return nil, err
	}

	// Verify borrower exists and belongs to the authenticated user
	p, err := s.peerRepo.GetByIDAndUser(ctx, req.PeerID, userID)
	if err != nil {
		if errors.Is(err, internalErrors.ErrNotFound) {
			return nil, internalErrors.NewValidationError("borrower contact not found", map[string]string{
				"peer_id": "the specified peer does not exist or does not belong to your account",
			})
		}
		return nil, err
	}

	newLoan := &Loan{
		UserID:             userID,
		PeerID:             req.PeerID,
		PeerName:           p.Name,
		PeerEmail:          p.Email,
		PrincipalAmount:    principal,
		Currency:           req.Currency,
		InterestType:       InterestType(req.InterestType),
		AnnualInterestRate: rate,
		DayCountConvention: "ACTUAL_365",
		StartDate:          startDate,
		DueDate:            dueDate,
		Status:             StatusDraft,
		Version:            1,
		Notes:              req.Notes,
	}

	created, err := s.repo.Create(ctx, newLoan)
	if err != nil {
		return nil, fmt.Errorf("failed to create loan: %w", err)
	}
	created.PeerName = p.Name
	created.PeerEmail = p.Email

	// Audit log
	payload, _ := json.Marshal(map[string]any{
		"principal": principal.StringFixed(2),
		"currency":  req.Currency,
		"interest":  req.InterestType,
		"rate":      rate.StringFixed(2),
		"peer_id":   req.PeerID.String(),
	})
	_, _ = s.repo.CreateLedgerEvent(ctx, &LedgerEvent{
		UserID:    userID,
		LoanID:    &created.ID,
		EventType: "LOAN_CREATED",
		ActorType: "LENDER",
		ActorID:   &userID,
		Payload:   payload,
	})

	summary := s.calculator.CalculateSummary(
		created.PrincipalAmount,
		string(created.InterestType),
		created.AnnualInterestRate,
		created.StartDate,
		time.Now().UTC(),
		decimal.Zero,
	)

	res := ToLoanResponse(created, summary)
	return &res, nil
}

func (s *loanService) GetByID(ctx context.Context, id, userID uuid.UUID) (*LoanResponse, error) {
	loan, err := s.repo.GetByIDAndUser(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	p, err := s.peerRepo.GetByIDAndUser(ctx, loan.PeerID, userID)
	if err == nil {
		loan.PeerName = p.Name
		loan.PeerEmail = p.Email
	}

	asOf := time.Now().UTC()
	repaid := s.getConfirmedRepayments(ctx, loan.ID, userID)
	summary := s.calculator.CalculateSummary(
		loan.PrincipalAmount,
		string(loan.InterestType),
		loan.AnnualInterestRate,
		loan.StartDate,
		asOf,
		repaid,
	)

	res := ToLoanResponse(loan, summary)
	return &res, nil
}

func (s *loanService) List(ctx context.Context, userID uuid.UUID, query ListLoansQuery) ([]LoanResponse, *server.Meta, error) {
	query.Sanitize()

	loans, total, err := s.repo.ListByUser(ctx, userID, query.PeerID, query.Status, query.Limit(), query.Offset())
	if err != nil {
		return nil, nil, err
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(query.PageSize)))
	}

	meta := &server.Meta{
		Page:       query.Page,
		PageSize:   query.PageSize,
		TotalItems: total,
		TotalPages: totalPages,
	}

	now := time.Now().UTC()
	responses := make([]LoanResponse, len(loans))
	for i, l := range loans {
		repaid := s.getConfirmedRepayments(ctx, l.ID, userID)
		summary := s.calculator.CalculateSummary(
			l.PrincipalAmount,
			string(l.InterestType),
			l.AnnualInterestRate,
			l.StartDate,
			now,
			repaid,
		)
		responses[i] = ToLoanResponse(l, summary)
	}

	return responses, meta, nil
}

func (s *loanService) UpdateDraft(ctx context.Context, id, userID uuid.UUID, req UpdateLoanRequest) (*LoanResponse, error) {
	existing, err := s.repo.GetByIDAndUser(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	if existing.Status != StatusDraft {
		return nil, internalErrors.NewConflictError("only draft loans can be edited; active loans must use the amendment workflow")
	}

	principal, rate, startDate, dueDate, err := req.SanitizeAndValidate()
	if err != nil {
		return nil, err
	}

	p, err := s.peerRepo.GetByIDAndUser(ctx, req.PeerID, userID)
	if err != nil {
		return nil, internalErrors.NewValidationError("borrower contact not found", map[string]string{
			"peer_id": "the specified peer does not exist or does not belong to your account",
		})
	}

	existing.PeerID = req.PeerID
	existing.PrincipalAmount = principal
	existing.Currency = req.Currency
	existing.InterestType = InterestType(req.InterestType)
	existing.AnnualInterestRate = rate
	existing.StartDate = startDate
	existing.DueDate = dueDate
	existing.Notes = req.Notes

	updated, err := s.repo.UpdateDraft(ctx, existing)
	if err != nil {
		return nil, err
	}
	updated.PeerName = p.Name
	updated.PeerEmail = p.Email

	repaid := s.getConfirmedRepayments(ctx, updated.ID, userID)
	summary := s.calculator.CalculateSummary(
		updated.PrincipalAmount,
		string(updated.InterestType),
		updated.AnnualInterestRate,
		updated.StartDate,
		time.Now().UTC(),
		repaid,
	)

	res := ToLoanResponse(updated, summary)
	return &res, nil
}

func (s *loanService) Activate(ctx context.Context, id, userID uuid.UUID) (*LoanResponse, error) {
	existing, err := s.repo.GetByIDAndUser(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	if existing.Status != StatusDraft {
		return nil, internalErrors.NewConflictError(fmt.Sprintf("cannot activate loan in %s status; only DRAFT loans can be activated", existing.Status))
	}

	updated, err := s.repo.UpdateStatus(ctx, id, userID, StatusActive)
	if err != nil {
		return nil, err
	}

	p, _ := s.peerRepo.GetByIDAndUser(ctx, updated.PeerID, userID)
	if p != nil {
		updated.PeerName = p.Name
		updated.PeerEmail = p.Email
	}

	// Audit log
	_, _ = s.repo.CreateLedgerEvent(ctx, &LedgerEvent{
		UserID:    userID,
		LoanID:    &id,
		EventType: "LOAN_ACTIVATED",
		ActorType: "LENDER",
		ActorID:   &userID,
		Payload:   json.RawMessage(`{"status":"ACTIVE"}`),
	})

	repaid := s.getConfirmedRepayments(ctx, updated.ID, userID)
	summary := s.calculator.CalculateSummary(
		updated.PrincipalAmount,
		string(updated.InterestType),
		updated.AnnualInterestRate,
		updated.StartDate,
		time.Now().UTC(),
		repaid,
	)

	res := ToLoanResponse(updated, summary)
	return &res, nil
}

func (s *loanService) Cancel(ctx context.Context, id, userID uuid.UUID, req CancelLoanRequest) (*LoanResponse, error) {
	existing, err := s.repo.GetByIDAndUser(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	if existing.Status == StatusCompleted || existing.Status == StatusCancelled {
		return nil, internalErrors.NewConflictError(fmt.Sprintf("cannot cancel loan in %s status", existing.Status))
	}

	updated, err := s.repo.UpdateStatus(ctx, id, userID, StatusCancelled)
	if err != nil {
		return nil, err
	}

	p, _ := s.peerRepo.GetByIDAndUser(ctx, updated.PeerID, userID)
	if p != nil {
		updated.PeerName = p.Name
		updated.PeerEmail = p.Email
	}

	// Audit log
	payload, _ := json.Marshal(map[string]any{
		"reason": req.Reason,
		"status": "CANCELLED",
	})
	_, _ = s.repo.CreateLedgerEvent(ctx, &LedgerEvent{
		UserID:    userID,
		LoanID:    &id,
		EventType: "LOAN_CANCELLED",
		ActorType: "LENDER",
		ActorID:   &userID,
		Payload:   payload,
	})

	repaid := s.getConfirmedRepayments(ctx, updated.ID, userID)
	summary := s.calculator.CalculateSummary(
		updated.PrincipalAmount,
		string(updated.InterestType),
		updated.AnnualInterestRate,
		updated.StartDate,
		time.Now().UTC(),
		repaid,
	)

	res := ToLoanResponse(updated, summary)
	return &res, nil
}

func (s *loanService) Dispute(ctx context.Context, id, userID uuid.UUID, req DisputeLoanRequest) (*LoanResponse, error) {
	existing, err := s.repo.GetByIDAndUser(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	if existing.Status != StatusActive {
		return nil, internalErrors.NewConflictError(fmt.Sprintf("cannot dispute loan in %s status; only ACTIVE loans can be disputed", existing.Status))
	}

	if req.Reason == "" {
		return nil, internalErrors.NewValidationError("dispute reason is required", map[string]string{
			"reason": "dispute reason cannot be blank",
		})
	}

	updated, err := s.repo.UpdateStatus(ctx, id, userID, StatusDisputed)
	if err != nil {
		return nil, err
	}

	p, _ := s.peerRepo.GetByIDAndUser(ctx, updated.PeerID, userID)
	if p != nil {
		updated.PeerName = p.Name
		updated.PeerEmail = p.Email
	}

	// Audit log
	payload, _ := json.Marshal(map[string]any{
		"reason": req.Reason,
		"status": "DISPUTED",
	})
	_, _ = s.repo.CreateLedgerEvent(ctx, &LedgerEvent{
		UserID:    userID,
		LoanID:    &id,
		EventType: "LOAN_DISPUTED",
		ActorType: "LENDER",
		ActorID:   &userID,
		Payload:   payload,
	})

	repaid := s.getConfirmedRepayments(ctx, updated.ID, userID)
	summary := s.calculator.CalculateSummary(
		updated.PrincipalAmount,
		string(updated.InterestType),
		updated.AnnualInterestRate,
		updated.StartDate,
		time.Now().UTC(),
		repaid,
	)

	res := ToLoanResponse(updated, summary)
	return &res, nil
}

func (s *loanService) Amend(ctx context.Context, id, userID uuid.UUID, req AmendLoanRequest) (*LoanResponse, error) {
	existing, err := s.repo.GetByIDAndUser(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	if existing.Status != StatusActive {
		return nil, internalErrors.NewConflictError(fmt.Sprintf("cannot amend loan in %s status; only ACTIVE loans can be amended", existing.Status))
	}

	principal, rate, dueDate, err := req.SanitizeAndValidate(existing.StartDate)
	if err != nil {
		return nil, err
	}

	updated, err := s.repo.Amend(ctx, id, userID, principal, rate, InterestType(req.InterestType), dueDate, req.Notes)
	if err != nil {
		return nil, err
	}

	p, _ := s.peerRepo.GetByIDAndUser(ctx, updated.PeerID, userID)
	if p != nil {
		updated.PeerName = p.Name
		updated.PeerEmail = p.Email
	}

	// Audit log
	payload, _ := json.Marshal(map[string]any{
		"reason":            req.Reason,
		"new_version":       updated.Version,
		"old_principal":     existing.PrincipalAmount.StringFixed(2),
		"new_principal":     updated.PrincipalAmount.StringFixed(2),
		"old_interest_type": existing.InterestType,
		"new_interest_type": updated.InterestType,
		"old_rate":          existing.AnnualInterestRate.StringFixed(2),
		"new_rate":          updated.AnnualInterestRate.StringFixed(2),
	})
	_, _ = s.repo.CreateLedgerEvent(ctx, &LedgerEvent{
		UserID:    userID,
		LoanID:    &id,
		EventType: "LOAN_AMENDED",
		ActorType: "LENDER",
		ActorID:   &userID,
		Payload:   payload,
	})

	repaid := s.getConfirmedRepayments(ctx, updated.ID, userID)
	summary := s.calculator.CalculateSummary(
		updated.PrincipalAmount,
		string(updated.InterestType),
		updated.AnnualInterestRate,
		updated.StartDate,
		time.Now().UTC(),
		repaid,
	)

	res := ToLoanResponse(updated, summary)
	return &res, nil
}

func (s *loanService) ListAuditEvents(ctx context.Context, id, userID uuid.UUID) ([]*LedgerEvent, error) {
	// Verify loan exists and belongs to the user
	if _, err := s.repo.GetByIDAndUser(ctx, id, userID); err != nil {
		return nil, err
	}

	return s.repo.ListLedgerEvents(ctx, id, userID)
}
