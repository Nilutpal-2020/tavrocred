package repayment

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Status string

const (
	StatusPendingVerification Status = "PENDING_VERIFICATION"
	StatusConfirmed           Status = "CONFIRMED"
	StatusRejected            Status = "REJECTED"
	StatusReversed            Status = "REVERSED"
)

type Source string

const (
	SourceLenderRecorded   Source = "LENDER_RECORDED"
	SourceBorrowerReported Source = "BORROWER_REPORTED"
)

type Repayment struct {
	ID                uuid.UUID       `json:"id"`
	UserID            uuid.UUID       `json:"user_id"`
	LoanID            uuid.UUID       `json:"loan_id"`
	Amount            decimal.Decimal `json:"amount"`
	Currency          string          `json:"currency"`
	PaymentDate       time.Time       `json:"payment_date"`
	PaymentMethod     string          `json:"payment_method"`
	ExternalReference *string         `json:"external_reference,omitempty"`
	Status            Status          `json:"status"`
	Source            Source          `json:"source"`
	ConfirmedAt       *time.Time      `json:"confirmed_at,omitempty"`
	ConfirmedBy       *uuid.UUID      `json:"confirmed_by,omitempty"`
	RejectionReason   *string         `json:"rejection_reason,omitempty"`
	ReversedAt        *time.Time      `json:"reversed_at,omitempty"`
	ReversalReason    *string         `json:"reversal_reason,omitempty"`
	Notes             *string         `json:"notes,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}
