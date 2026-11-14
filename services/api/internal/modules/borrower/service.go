package borrower

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	internalErrors "tavrocred/services/api/internal/errors"
	"tavrocred/services/api/internal/modules/auth"
	"tavrocred/services/api/internal/modules/loan"
	"tavrocred/services/api/internal/modules/repayment"
)

type Service interface {
	GenerateBorrowerLink(ctx context.Context, loanID, userID uuid.UUID, expiresIn time.Duration) (*BorrowerLinkResponse, error)
	RevokeBorrowerLink(ctx context.Context, loanID, userID uuid.UUID) error
	ExchangeTokenForSession(ctx context.Context, rawToken string) (*BorrowerAccessResponse, error)
	GetLoanDetailsForBorrower(ctx context.Context, loanID, peerID uuid.UUID) (*BorrowerLoanViewResponse, error)
	AcknowledgeLoan(ctx context.Context, loanID, peerID uuid.UUID, ip, userAgent string) (*LoanAcknowledgement, error)
	DisputeLoan(ctx context.Context, loanID, peerID uuid.UUID, reason, ip, userAgent string) (*LoanAcknowledgement, error)
	ReportRepayment(ctx context.Context, loanID, peerID uuid.UUID, req BorrowerReportRepaymentRequest) (*repayment.RepaymentResponse, error)
}

type borrowerService struct {
	repo          Repository
	loanRepo      loan.Repository
	repaymentRepo repayment.Repository
	authRepo      auth.Repository
	calculator    *loan.Calculator
	tokenManager  TokenManager
	appBaseURL    string
}

func NewService(
	repo Repository,
	loanRepo loan.Repository,
	repaymentRepo repayment.Repository,
	authRepo auth.Repository,
	calculator *loan.Calculator,
	tokenManager TokenManager,
	appBaseURL string,
) Service {
	if appBaseURL == "" {
		appBaseURL = "http://localhost:3000"
	}
	return &borrowerService{
		repo:          repo,
		loanRepo:      loanRepo,
		repaymentRepo: repaymentRepo,
		authRepo:      authRepo,
		calculator:    calculator,
		tokenManager:  tokenManager,
		appBaseURL:    appBaseURL,
	}
}

func (s *borrowerService) GenerateBorrowerLink(ctx context.Context, loanID, userID uuid.UUID, expiresIn time.Duration) (*BorrowerLinkResponse, error) {
	l, err := s.loanRepo.GetByIDAndUser(ctx, loanID, userID)
	if err != nil {
		return nil, err
	}

	if l.Status == loan.StatusDraft || l.Status == loan.StatusCancelled {
		return nil, internalErrors.NewConflictError(fmt.Sprintf("cannot generate borrower link for loan in %s status", l.Status))
	}

	// Revoke existing active borrower access tokens for this loan
	_ = s.repo.RevokeAccessTokensByLoan(ctx, loanID, userID)

	rawToken, tokenHash, err := s.tokenManager.GenerateRawToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate secure token: %w", err)
	}

	expiresAt := time.Now().UTC().Add(expiresIn)
	_, err = s.repo.CreateAccessToken(ctx, &BorrowerAccessToken{
		LoanID:    loanID,
		PeerID:    l.PeerID,
		UserID:    userID,
		TokenHash: tokenHash,
		Purpose:   PurposePortalAccess,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to persist borrower access token: %w", err)
	}

	url := fmt.Sprintf("%s/portal/loans?token=%s", s.appBaseURL, rawToken)
	return &BorrowerLinkResponse{
		Token:     rawToken,
		URL:       url,
		ExpiresAt: expiresAt,
	}, nil
}

func (s *borrowerService) RevokeBorrowerLink(ctx context.Context, loanID, userID uuid.UUID) error {
	// Verify loan ownership
	if _, err := s.loanRepo.GetByIDAndUser(ctx, loanID, userID); err != nil {
		return err
	}

	return s.repo.RevokeAccessTokensByLoan(ctx, loanID, userID)
}

func (s *borrowerService) ExchangeTokenForSession(ctx context.Context, rawToken string) (*BorrowerAccessResponse, error) {
	tokenHash := s.tokenManager.HashToken(rawToken)

	tok, err := s.repo.GetActiveAccessTokenByHash(ctx, tokenHash)
	if err != nil {
		return nil, internalErrors.NewUnauthorizedError("invalid, expired, or revoked borrower access link")
	}

	// Session duration: 4 hours
	sessionDuration := 4 * time.Hour
	sessionToken, err := s.tokenManager.SignSessionToken(tok.LoanID, tok.PeerID, tok.UserID, sessionDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to sign borrower session: %w", err)
	}

	return &BorrowerAccessResponse{
		SessionToken: sessionToken,
		LoanID:       tok.LoanID,
		ExpiresAt:    time.Now().UTC().Add(sessionDuration),
	}, nil
}

func (s *borrowerService) GetLoanDetailsForBorrower(ctx context.Context, loanID, peerID uuid.UUID) (*BorrowerLoanViewResponse, error) {
	l, err := s.repo.GetLoanByIDAndPeer(ctx, loanID, peerID)
	if err != nil {
		return nil, err
	}

	lenderUser, err := s.authRepo.GetUserByID(ctx, l.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve lender details: %w", err)
	}

	now := time.Now().UTC()
	confirmedTotal, err := s.repaymentRepo.GetConfirmedTotalByLoan(ctx, loanID, l.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get confirmed total: %w", err)
	}

	summary := s.calculator.CalculateSummary(
		l.PrincipalAmount,
		string(l.InterestType),
		l.AnnualInterestRate,
		l.StartDate,
		now,
		confirmedTotal,
	)

	repayments, _, err := s.repaymentRepo.ListByLoan(ctx, loanID, l.UserID, "", 100, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to list repayments: %w", err)
	}

	repaymentResponses := make([]repayment.RepaymentResponse, len(repayments))
	for i, r := range repayments {
		repaymentResponses[i] = repayment.ToRepaymentResponse(r)
	}

	var latestAckInfo *BorrowerAcknowledgementInfo
	ack, err := s.repo.GetLatestAcknowledgementByLoan(ctx, loanID)
	if err == nil && ack != nil {
		latestAckInfo = &BorrowerAcknowledgementInfo{
			Version:       ack.Version,
			Status:        string(ack.Status),
			DisputeReason: ack.DisputeReason,
			RespondedAt:   ack.RespondedAt,
		}
	}

	var dueDateStr *string
	if l.DueDate != nil {
		s := l.DueDate.Format("2006-01-02")
		dueDateStr = &s
	}

	return &BorrowerLoanViewResponse{
		Loan: BorrowerLoanDetail{
			ID:                 l.ID,
			Currency:           l.Currency,
			PrincipalAmount:    l.PrincipalAmount.StringFixed(2),
			InterestType:       string(l.InterestType),
			AnnualInterestRate: l.AnnualInterestRate.StringFixed(2),
			DayCountConvention: l.DayCountConvention,
			StartDate:          l.StartDate.Format("2006-01-02"),
			DueDate:            dueDateStr,
			Status:             string(l.Status),
			Version:            l.Version,
			Notes:              l.Notes,
		},
		Summary: summary,
		Lender: BorrowerLenderInfo{
			Name:  lenderUser.FullName,
			Email: lenderUser.Email,
		},
		LatestAcknowledgement: latestAckInfo,
		Repayments:            repaymentResponses,
	}, nil
}

func (s *borrowerService) AcknowledgeLoan(ctx context.Context, loanID, peerID uuid.UUID, ip, userAgent string) (*LoanAcknowledgement, error) {
	l, err := s.repo.GetLoanByIDAndPeer(ctx, loanID, peerID)
	if err != nil {
		return nil, err
	}

	if l.Status != loan.StatusActive && l.Status != loan.StatusDisputed {
		return nil, internalErrors.NewConflictError(fmt.Sprintf("cannot acknowledge loan in %s status", l.Status))
	}

	now := time.Now().UTC()
	var ipPtr *string
	if ip != "" {
		ipPtr = &ip
	}
	var uaPtr *string
	if userAgent != "" {
		uaPtr = &userAgent
	}

	ack := &LoanAcknowledgement{
		LoanID:      loanID,
		PeerID:      peerID,
		Version:     l.Version,
		Status:      StatusAcknowledged,
		IPAddress:   ipPtr,
		UserAgent:   uaPtr,
		RespondedAt: now,
	}

	created, err := s.repo.CreateAcknowledgement(ctx, ack)
	if err != nil {
		return nil, fmt.Errorf("failed to record acknowledgement: %w", err)
	}

	// If loan was disputed, acknowledge clears dispute back to active
	if l.Status == loan.StatusDisputed {
		_, _ = s.repo.UpdateLoanStatusByPeer(ctx, loanID, peerID, loan.StatusActive)
	}

	// Append-only audit log: BORROWER_ACKNOWLEDGED
	payload, _ := json.Marshal(map[string]any{
		"loan_id":    loanID.String(),
		"peer_id":    peerID.String(),
		"version":    l.Version,
		"status":     "ACKNOWLEDGED",
		"timestamp":  now.Format(time.RFC3339),
		"ip_address": ip,
	})
	_, _ = s.loanRepo.CreateLedgerEvent(ctx, &loan.LedgerEvent{
		UserID:    l.UserID,
		LoanID:    &loanID,
		EventType: "BORROWER_ACKNOWLEDGED",
		ActorType: "BORROWER",
		ActorID:   &peerID,
		Payload:   payload,
	})

	return created, nil
}

func (s *borrowerService) DisputeLoan(ctx context.Context, loanID, peerID uuid.UUID, reason, ip, userAgent string) (*LoanAcknowledgement, error) {
	l, err := s.repo.GetLoanByIDAndPeer(ctx, loanID, peerID)
	if err != nil {
		return nil, err
	}

	if l.Status != loan.StatusActive && l.Status != loan.StatusDisputed {
		return nil, internalErrors.NewConflictError(fmt.Sprintf("cannot dispute loan in %s status", l.Status))
	}

	if reason == "" {
		return nil, internalErrors.NewValidationError("dispute reason is required", map[string]string{
			"reason": "dispute reason cannot be blank",
		})
	}

	now := time.Now().UTC()
	var ipPtr *string
	if ip != "" {
		ipPtr = &ip
	}
	var uaPtr *string
	if userAgent != "" {
		uaPtr = &userAgent
	}

	ack := &LoanAcknowledgement{
		LoanID:        loanID,
		PeerID:        peerID,
		Version:       l.Version,
		Status:        StatusDisputed,
		DisputeReason: &reason,
		IPAddress:     ipPtr,
		UserAgent:     uaPtr,
		RespondedAt:   now,
	}

	created, err := s.repo.CreateAcknowledgement(ctx, ack)
	if err != nil {
		return nil, fmt.Errorf("failed to record dispute: %w", err)
	}

	// Update loan status to DISPUTED
	_, err = s.repo.UpdateLoanStatusByPeer(ctx, loanID, peerID, loan.StatusDisputed)
	if err != nil {
		return nil, fmt.Errorf("failed to update loan status: %w", err)
	}

	// Append-only audit log: LOAN_DISPUTED
	payload, _ := json.Marshal(map[string]any{
		"loan_id":    loanID.String(),
		"peer_id":    peerID.String(),
		"version":    l.Version,
		"status":     "DISPUTED",
		"reason":     reason,
		"timestamp":  now.Format(time.RFC3339),
		"ip_address": ip,
	})
	_, _ = s.loanRepo.CreateLedgerEvent(ctx, &loan.LedgerEvent{
		UserID:    l.UserID,
		LoanID:    &loanID,
		EventType: "LOAN_DISPUTED",
		ActorType: "BORROWER",
		ActorID:   &peerID,
		Payload:   payload,
	})

	return created, nil
}

func (s *borrowerService) ReportRepayment(ctx context.Context, loanID, peerID uuid.UUID, req BorrowerReportRepaymentRequest) (*repayment.RepaymentResponse, error) {
	l, err := s.repo.GetLoanByIDAndPeer(ctx, loanID, peerID)
	if err != nil {
		return nil, err
	}

	if l.Status == loan.StatusDraft || l.Status == loan.StatusCancelled || l.Status == loan.StatusCompleted {
		return nil, internalErrors.NewConflictError(fmt.Sprintf("cannot report repayment for loan in %s status", l.Status))
	}

	amount, payDate, err := req.SanitizeAndValidate(l.StartDate)
	if err != nil {
		return nil, err
	}

	confirmedTotal, err := s.repaymentRepo.GetConfirmedTotalByLoan(ctx, loanID, l.UserID)
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
			"amount": fmt.Sprintf("reported repayment amount (%s) exceeds current outstanding balance (%s)", amount.StringFixed(2), summary.OutstandingBalance.StringFixed(2)),
		})
	}

	rep := &repayment.Repayment{
		UserID:            l.UserID,
		LoanID:            loanID,
		Amount:            amount,
		Currency:          l.Currency,
		PaymentDate:       payDate,
		PaymentMethod:     req.PaymentMethod,
		ExternalReference: req.ExternalReference,
		Status:            repayment.StatusPendingVerification,
		Source:            repayment.SourceBorrowerReported,
		Notes:             req.Notes,
	}

	created, err := s.repaymentRepo.Create(ctx, rep)
	if err != nil {
		return nil, fmt.Errorf("failed to record borrower-reported repayment: %w", err)
	}

	// Ledger Event: REPAYMENT_REPORTED
	payload, _ := json.Marshal(map[string]any{
		"repayment_id":   created.ID.String(),
		"amount":         amount.StringFixed(2),
		"currency":       l.Currency,
		"payment_date":   payDate.Format("2006-01-02"),
		"payment_method": req.PaymentMethod,
		"source":         "BORROWER_REPORTED",
	})
	_, _ = s.loanRepo.CreateLedgerEvent(ctx, &loan.LedgerEvent{
		UserID:    l.UserID,
		LoanID:    &loanID,
		EventType: "REPAYMENT_REPORTED",
		ActorType: "BORROWER",
		ActorID:   &peerID,
		Payload:   payload,
	})

	res := repayment.ToRepaymentResponse(created)
	return &res, nil
}
