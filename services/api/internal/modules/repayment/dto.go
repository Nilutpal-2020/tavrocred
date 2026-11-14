package repayment

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	internalErrors "tavrocred/services/api/internal/errors"
)

type RecordRepaymentRequest struct {
	Amount            string  `json:"amount"`
	Currency          string  `json:"currency"`
	PaymentDate       string  `json:"payment_date"`
	PaymentMethod     string  `json:"payment_method"`
	ExternalReference *string `json:"external_reference,omitempty"`
	Notes             *string `json:"notes,omitempty"`
	ConfirmedDirectly bool    `json:"confirmed_directly"`
}

func (r *RecordRepaymentRequest) SanitizeAndValidate(loanStartDate time.Time) (decimal.Decimal, time.Time, error) {
	details := make(map[string]string)

	amount, err := decimal.NewFromString(strings.TrimSpace(r.Amount))
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		details["amount"] = "amount must be a valid positive number"
	}

	r.Currency = strings.ToUpper(strings.TrimSpace(r.Currency))
	if r.Currency == "" {
		r.Currency = "INR"
	} else if len(r.Currency) != 3 {
		details["currency"] = "currency must be a 3-letter ISO code (e.g. INR)"
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
		// Allow up to tomorrow to handle UTC / local timezone shifts
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
		return decimal.Zero, time.Time{}, internalErrors.NewValidationError("invalid repayment data", details)
	}

	return amount.Round(2), payDate, nil
}

type RejectRepaymentRequest struct {
	Reason string `json:"reason"`
}

func (r *RejectRepaymentRequest) Validate() error {
	r.Reason = strings.TrimSpace(r.Reason)
	if r.Reason == "" {
		return internalErrors.NewValidationError("rejection reason is required", map[string]string{
			"reason": "reason cannot be blank",
		})
	}
	return nil
}

type ReverseRepaymentRequest struct {
	Reason string `json:"reason"`
}

func (r *ReverseRepaymentRequest) Validate() error {
	r.Reason = strings.TrimSpace(r.Reason)
	if r.Reason == "" {
		return internalErrors.NewValidationError("reversal reason is required", map[string]string{
			"reason": "reason cannot be blank",
		})
	}
	return nil
}

type RepaymentResponse struct {
	ID                uuid.UUID  `json:"id"`
	UserID            uuid.UUID  `json:"user_id"`
	LoanID            uuid.UUID  `json:"loan_id"`
	Amount            string     `json:"amount"`
	Currency          string     `json:"currency"`
	PaymentDate       string     `json:"payment_date"`
	PaymentMethod     string     `json:"payment_method"`
	ExternalReference *string    `json:"external_reference,omitempty"`
	Status            string     `json:"status"`
	Source            string     `json:"source"`
	ConfirmedAt       *time.Time `json:"confirmed_at,omitempty"`
	ConfirmedBy       *uuid.UUID `json:"confirmed_by,omitempty"`
	RejectionReason   *string    `json:"rejection_reason,omitempty"`
	ReversedAt        *time.Time `json:"reversed_at,omitempty"`
	ReversalReason    *string    `json:"reversal_reason,omitempty"`
	Notes             *string    `json:"notes,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func ToRepaymentResponse(r *Repayment) RepaymentResponse {
	return RepaymentResponse{
		ID:                r.ID,
		UserID:            r.UserID,
		LoanID:            r.LoanID,
		Amount:            r.Amount.StringFixed(2),
		Currency:          r.Currency,
		PaymentDate:       r.PaymentDate.Format("2006-01-02"),
		PaymentMethod:     r.PaymentMethod,
		ExternalReference: r.ExternalReference,
		Status:            string(r.Status),
		Source:            string(r.Source),
		ConfirmedAt:       r.ConfirmedAt,
		ConfirmedBy:       r.ConfirmedBy,
		RejectionReason:   r.RejectionReason,
		ReversedAt:        r.ReversedAt,
		ReversalReason:    r.ReversalReason,
		Notes:             r.Notes,
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
	}
}

type ListRepaymentsQuery struct {
	Page     int
	PageSize int
	Status   string
}

func (q *ListRepaymentsQuery) Sanitize() {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = 20
	} else if q.PageSize > 100 {
		q.PageSize = 100
	}
	q.Status = strings.ToUpper(strings.TrimSpace(q.Status))
}

func (q *ListRepaymentsQuery) Offset() int32 {
	return int32((q.Page - 1) * q.PageSize)
}

func (q *ListRepaymentsQuery) Limit() int32 {
	return int32(q.PageSize)
}
