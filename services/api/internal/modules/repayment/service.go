package repayment

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	internalErrors "tavrocred/services/api/internal/errors"
	"tavrocred/services/api/internal/modules/loan"
	"tavrocred/services/api/internal/server"
)

type Service interface {
	Record(ctx context.Context, loanID, userID uuid.UUID, req RecordRepaymentRequest) (*RepaymentResponse, error)
	GetByID(ctx context.Context, id, userID uuid.UUID) (*RepaymentResponse, error)
	ListByLoan(ctx context.Context, loanID, userID uuid.UUID, query ListRepaymentsQuery) ([]RepaymentResponse, *server.Meta, error)
	Confirm(ctx context.Context, id, userID uuid.UUID) (*RepaymentResponse, error)
	Reject(ctx context.Context, id, userID uuid.UUID, req RejectRepaymentRequest) (*RepaymentResponse, error)
	Reverse(ctx context.Context, id, userID uuid.UUID, req ReverseRepaymentRequest) (*RepaymentResponse, error)
}

type repaymentService struct {
	repo       Repository
	loanRepo   loan.Repository
	calculator *loan.Calculator
}

func NewService(repo Repository, loanRepo loan.Repository, calculator *loan.Calculator) Service {
	return &repaymentService{
		repo:       repo,
		loanRepo:   loanRepo,
		calculator: calculator,
	}
}

func (s *repaymentService) Record(ctx context.Context, loanID, userID uuid.UUID, req RecordRepaymentRequest) (*RepaymentResponse, error) {
	l, err := s.loanRepo.GetByIDAndUser(ctx, loanID, userID)
	if err != nil {
		return nil, err
	}

	if l.Status == loan.StatusDraft || l.Status == loan.StatusCancelled {
		return nil, internalErrors.NewConflictError(fmt.Sprintf("cannot record repayment for loan in %s status", l.Status))
	}

	amount, payDate, err := req.SanitizeAndValidate(l.StartDate)
	if err != nil {
		return nil, err
	}

	status := StatusPendingVerification
	var confirmedAt *time.Time
	var confirmedBy *uuid.UUID

	if req.ConfirmedDirectly {
		confirmedTotal, err := s.repo.GetConfirmedTotalByLoan(ctx, loanID, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to get confirmed total: %w", err)
		}

		summary := s.calculator.CalculateSummary(
			l.PrincipalAmount,
			string(l.InterestType),
			l.AnnualInterestRate,
			l.StartDate,
			payDate,
			confirmedTotal,
		)

		if amount.GreaterThan(summary.OutstandingBalance) {
			return nil, internalErrors.NewValidationError("overpayment rejected", map[string]string{
				"amount": fmt.Sprintf("repayment amount (%s) exceeds current outstanding balance (%s)", amount.StringFixed(2), summary.OutstandingBalance.StringFixed(2)),
			})
		}

		status = StatusConfirmed
		now := time.Now().UTC()
		confirmedAt = &now
		confirmedBy = &userID
	}

	rep := &Repayment{
		UserID:            userID,
		LoanID:            loanID,
		Amount:            amount,
		Currency:          req.Currency,
		PaymentDate:       payDate,
		PaymentMethod:     req.PaymentMethod,
		ExternalReference: req.ExternalReference,
		Status:            status,
		Source:            SourceLenderRecorded,
		ConfirmedAt:       confirmedAt,
		ConfirmedBy:       confirmedBy,
		Notes:             req.Notes,
	}

	created, err := s.repo.Create(ctx, rep)
	if err != nil {
		return nil, fmt.Errorf("failed to record repayment: %w", err)
	}

	// Ledger Event: REPAYMENT_RECORDED
	recPayload, _ := json.Marshal(map[string]any{
		"repayment_id":       created.ID.String(),
		"amount":             amount.StringFixed(2),
		"currency":           req.Currency,
		"payment_date":       payDate.Format("2006-01-02"),
		"payment_method":     req.PaymentMethod,
		"confirmed_directly": req.ConfirmedDirectly,
	})
	_, _ = s.loanRepo.CreateLedgerEvent(ctx, &loan.LedgerEvent{
		UserID:    userID,
		LoanID:    &loanID,
		EventType: "REPAYMENT_RECORDED",
		ActorType: "LENDER",
		ActorID:   &userID,
		Payload:   recPayload,
	})

	if req.ConfirmedDirectly {
		// Ledger Event: REPAYMENT_CONFIRMED
		_, _ = s.loanRepo.CreateLedgerEvent(ctx, &loan.LedgerEvent{
			UserID:    userID,
			LoanID:    &loanID,
			EventType: "REPAYMENT_CONFIRMED",
			ActorType: "LENDER",
			ActorID:   &userID,
			Payload:   recPayload,
		})

		s.checkAndCompleteLoan(ctx, l, userID)
	}

	res := ToRepaymentResponse(created)
	return &res, nil
}

func (s *repaymentService) GetByID(ctx context.Context, id, userID uuid.UUID) (*RepaymentResponse, error) {
	rep, err := s.repo.GetByIDAndUser(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	res := ToRepaymentResponse(rep)
	return &res, nil
}

func (s *repaymentService) ListByLoan(ctx context.Context, loanID, userID uuid.UUID, query ListRepaymentsQuery) ([]RepaymentResponse, *server.Meta, error) {
	// Verify loan belongs to user
	if _, err := s.loanRepo.GetByIDAndUser(ctx, loanID, userID); err != nil {
		return nil, nil, err
	}

	query.Sanitize()

	reps, total, err := s.repo.ListByLoan(ctx, loanID, userID, query.Status, query.Limit(), query.Offset())
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

	responses := make([]RepaymentResponse, len(reps))
	for i, r := range reps {
		responses[i] = ToRepaymentResponse(r)
	}

	return responses, meta, nil
}

func (s *repaymentService) Confirm(ctx context.Context, id, userID uuid.UUID) (*RepaymentResponse, error) {
	rep, err := s.repo.GetByIDAndUser(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	if rep.Status != StatusPendingVerification {
		return nil, internalErrors.NewConflictError(fmt.Sprintf("cannot confirm repayment in %s status", rep.Status))
	}

	l, err := s.loanRepo.GetByIDAndUser(ctx, rep.LoanID, userID)
	if err != nil {
		return nil, err
	}

	confirmedTotal, err := s.repo.GetConfirmedTotalByLoan(ctx, rep.LoanID, userID)
	if err != nil {
		return nil, err
	}

	summary := s.calculator.CalculateSummary(
		l.PrincipalAmount,
		string(l.InterestType),
		l.AnnualInterestRate,
		l.StartDate,
		rep.PaymentDate,
		confirmedTotal,
	)

	if rep.Amount.GreaterThan(summary.OutstandingBalance) {
		return nil, internalErrors.NewValidationError("overpayment rejected", map[string]string{
			"amount": fmt.Sprintf("repayment amount (%s) exceeds current outstanding balance (%s)", rep.Amount.StringFixed(2), summary.OutstandingBalance.StringFixed(2)),
		})
	}

	confirmed, err := s.repo.Confirm(ctx, id, userID, userID)
	if err != nil {
		return nil, err
	}

	// Ledger Event: REPAYMENT_CONFIRMED
	payload, _ := json.Marshal(map[string]any{
		"repayment_id": confirmed.ID.String(),
		"amount":       confirmed.Amount.StringFixed(2),
		"currency":     confirmed.Currency,
	})
	_, _ = s.loanRepo.CreateLedgerEvent(ctx, &loan.LedgerEvent{
		UserID:    userID,
		LoanID:    &confirmed.LoanID,
		EventType: "REPAYMENT_CONFIRMED",
		ActorType: "LENDER",
		ActorID:   &userID,
		Payload:   payload,
	})

	s.checkAndCompleteLoan(ctx, l, userID)

	res := ToRepaymentResponse(confirmed)
	return &res, nil
}

func (s *repaymentService) Reject(ctx context.Context, id, userID uuid.UUID, req RejectRepaymentRequest) (*RepaymentResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	rep, err := s.repo.GetByIDAndUser(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	if rep.Status != StatusPendingVerification {
		return nil, internalErrors.NewConflictError(fmt.Sprintf("cannot reject repayment in %s status", rep.Status))
	}

	rejected, err := s.repo.Reject(ctx, id, userID, req.Reason)
	if err != nil {
		return nil, err
	}

	// Ledger Event: REPAYMENT_REJECTED
	payload, _ := json.Marshal(map[string]any{
		"repayment_id": rejected.ID.String(),
		"amount":       rejected.Amount.StringFixed(2),
		"reason":       req.Reason,
	})
	_, _ = s.loanRepo.CreateLedgerEvent(ctx, &loan.LedgerEvent{
		UserID:    userID,
		LoanID:    &rejected.LoanID,
		EventType: "REPAYMENT_REJECTED",
		ActorType: "LENDER",
		ActorID:   &userID,
		Payload:   payload,
	})

	res := ToRepaymentResponse(rejected)
	return &res, nil
}

func (s *repaymentService) Reverse(ctx context.Context, id, userID uuid.UUID, req ReverseRepaymentRequest) (*RepaymentResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	rep, err := s.repo.GetByIDAndUser(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	if rep.Status != StatusConfirmed {
		return nil, internalErrors.NewConflictError(fmt.Sprintf("cannot reverse repayment in %s status; only CONFIRMED repayments can be reversed", rep.Status))
	}

	reversed, err := s.repo.Reverse(ctx, id, userID, req.Reason)
	if err != nil {
		return nil, err
	}

	// Ledger Event: REPAYMENT_REVERSED
	payload, _ := json.Marshal(map[string]any{
		"repayment_id": reversed.ID.String(),
		"amount":       reversed.Amount.StringFixed(2),
		"reason":       req.Reason,
	})
	_, _ = s.loanRepo.CreateLedgerEvent(ctx, &loan.LedgerEvent{
		UserID:    userID,
		LoanID:    &reversed.LoanID,
		EventType: "REPAYMENT_REVERSED",
		ActorType: "LENDER",
		ActorID:   &userID,
		Payload:   payload,
	})

	// If loan was completed, restore it to ACTIVE status
	l, err := s.loanRepo.GetByIDAndUser(ctx, reversed.LoanID, userID)
	if err == nil && l.Status == loan.StatusCompleted {
		_, _ = s.loanRepo.UpdateStatus(ctx, l.ID, userID, loan.StatusActive)
		_, _ = s.loanRepo.CreateLedgerEvent(ctx, &loan.LedgerEvent{
			UserID:    userID,
			LoanID:    &l.ID,
			EventType: "LOAN_ACTIVATED",
			ActorType: "LENDER",
			ActorID:   &userID,
			Payload:   json.RawMessage(`{"reason":"reopened due to repayment reversal"}`),
		})
	}

	res := ToRepaymentResponse(reversed)
	return &res, nil
}

func (s *repaymentService) checkAndCompleteLoan(ctx context.Context, l *loan.Loan, userID uuid.UUID) {
	confirmedTotal, err := s.repo.GetConfirmedTotalByLoan(ctx, l.ID, userID)
	if err != nil {
		return
	}

	now := time.Now().UTC()
	summary := s.calculator.CalculateSummary(
		l.PrincipalAmount,
		string(l.InterestType),
		l.AnnualInterestRate,
		l.StartDate,
		now,
		confirmedTotal,
	)

	if summary.OutstandingBalance.LessThanOrEqual(decimal.Zero) && l.Status != loan.StatusCompleted {
		_, _ = s.loanRepo.UpdateStatus(ctx, l.ID, userID, loan.StatusCompleted)
		_, _ = s.loanRepo.CreateLedgerEvent(ctx, &loan.LedgerEvent{
			UserID:    userID,
			LoanID:    &l.ID,
			EventType: "LOAN_COMPLETED",
			ActorType: "LENDER",
			ActorID:   &userID,
			Payload:   json.RawMessage(`{"status":"COMPLETED","reason":"loan fully paid"}`),
		})
	}
}
