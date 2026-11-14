package borrower

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	internalErrors "tavrocred/services/api/internal/errors"
	"tavrocred/services/api/internal/modules/loan"
	"tavrocred/services/api/internal/modules/repayment"
)

type CreateBorrowerLinkRequest struct {
	ExpiresInHours int `json:"expires_in_hours"`
}

func (r *CreateBorrowerLinkRequest) Sanitize() time.Duration {
	hours := r.ExpiresInHours
	if hours < 1 {
		hours = 72 // 3 days default
	} else if hours > 720 {
		hours = 720 // 30 days max
	}
	return time.Duration(hours) * time.Hour
}

type BorrowerLinkResponse struct {
	Token     string    `json:"token"`
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

type BorrowerAccessRequest struct {
	Token string `json:"token"`
}

func (r *BorrowerAccessRequest) Validate() error {
	r.Token = strings.TrimSpace(r.Token)
	if len(r.Token) < 32 {
		return internalErrors.NewValidationError("invalid access token", map[string]string{
			"token": "token must be a valid secure token",
		})
	}
	return nil
}

type BorrowerAccessResponse struct {
	SessionToken string    `json:"session_token"`
	LoanID       uuid.UUID `json:"loan_id"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type BorrowerLoanViewResponse struct {
	Loan                  BorrowerLoanDetail           `json:"loan"`
	Summary               loan.FinancialSummary        `json:"summary"`
	Lender                BorrowerLenderInfo           `json:"lender"`
	LatestAcknowledgement *BorrowerAcknowledgementInfo `json:"latest_acknowledgement,omitempty"`
	Repayments            []repayment.RepaymentResponse `json:"repayments"`
}

type BorrowerLoanDetail struct {
	ID                 uuid.UUID `json:"id"`
	Currency           string    `json:"currency"`
	PrincipalAmount    string    `json:"principal_amount"`
	InterestType       string    `json:"interest_type"`
	AnnualInterestRate string    `json:"annual_interest_rate"`
	DayCountConvention string    `json:"day_count_convention"`
	StartDate          string    `json:"start_date"`
	DueDate            *string   `json:"due_date,omitempty"`
	Status             string    `json:"status"`
	Version            int       `json:"version"`
	Notes              *string   `json:"notes,omitempty"`
}

type BorrowerLenderInfo struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type BorrowerAcknowledgementInfo struct {
	Version       int       `json:"version"`
	Status        string    `json:"status"`
	DisputeReason *string   `json:"dispute_reason,omitempty"`
	RespondedAt   time.Time `json:"responded_at"`
}

type BorrowerDisputeRequest struct {
	Reason string `json:"reason"`
}

func (r *BorrowerDisputeRequest) Validate() error {
	r.Reason = strings.TrimSpace(r.Reason)
	if r.Reason == "" {
		return internalErrors.NewValidationError("dispute reason is required", map[string]string{
			"reason": "dispute reason cannot be blank",
		})
	}
	return nil
}

type BorrowerReportRepaymentRequest struct {
	Amount            string  `json:"amount"`
	PaymentDate       string  `json:"payment_date"`
	PaymentMethod     string  `json:"payment_method"`
	ExternalReference *string `json:"external_reference,omitempty"`
	Notes             *string `json:"notes,omitempty"`
}

func (r *BorrowerReportRepaymentRequest) SanitizeAndValidate(loanStartDate time.Time) (decimal.Decimal, time.Time, error) {
	details := make(map[string]string)

	amount, err := decimal.NewFromString(strings.TrimSpace(r.Amount))
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		details["amount"] = "amount must be a valid positive number"
	}

	r.PaymentMethod = strings.ToUpper(strings.TrimSpace(r.PaymentMethod))
	validMethods := map[string]bool{
		"UPI":           true,
		"BANK_TRANSFER": true,
		"CASH":          true,
		"OTHER":         true,
	}
	if !validMethods[r.PaymentMethod] {
		details["payment_method"] = "payment_method must be one of: UPI, BANK_TRANSFER, CASH, OTHER"
	}

	payDate, err := time.Parse("2006-01-02", strings.TrimSpace(r.PaymentDate))
	if err != nil {
		details["payment_date"] = "payment_date must be in YYYY-MM-DD format"
	} else {
		loanStartCal := time.Date(loanStartDate.Year(), loanStartDate.Month(), loanStartDate.Day(), 0, 0, 0, 0, time.UTC)
		if payDate.Before(loanStartCal) {
			details["payment_date"] = "payment_date cannot be earlier than loan start_date"
		}
		tomorrow := time.Now().UTC().Add(24 * time.Hour)
		if payDate.After(tomorrow) {
			details["payment_date"] = "payment_date cannot be in the future"
		}
	}

	if r.ExternalReference != nil {
		clean := strings.TrimSpace(*r.ExternalReference)
		if clean == "" {
			r.ExternalReference = nil
		} else if len(clean) > 100 {
			details["external_reference"] = "external_reference cannot exceed 100 characters"
		} else {
			r.ExternalReference = &clean
		}
	}

	if r.Notes != nil {
		clean := strings.TrimSpace(*r.Notes)
		if clean == "" {
			r.Notes = nil
		} else {
			r.Notes = &clean
		}
	}

	if len(details) > 0 {
		return decimal.Zero, time.Time{}, internalErrors.NewValidationError("invalid reported repayment data", details)
	}

	return amount.Round(2), payDate, nil
}
