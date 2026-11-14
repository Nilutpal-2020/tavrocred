package loan

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Status string

const (
	StatusDraft     Status = "DRAFT"
	StatusActive    Status = "ACTIVE"
	StatusCompleted Status = "COMPLETED"
	StatusCancelled Status = "CANCELLED"
	StatusDisputed  Status = "DISPUTED"
)

type InterestType string

const (
	InterestTypeNone   InterestType = "NONE"
	InterestTypeSimple InterestType = "SIMPLE"
)

type Loan struct {
	ID                 uuid.UUID       `json:"id"`
	UserID             uuid.UUID       `json:"user_id"`
	PeerID             uuid.UUID       `json:"peer_id"`
	PeerName           string          `json:"peer_name,omitempty"`
	PeerEmail          *string         `json:"peer_email,omitempty"`
	PrincipalAmount    decimal.Decimal `json:"principal_amount"`
	Currency           string          `json:"currency"`
	InterestType       InterestType    `json:"interest_type"`
	AnnualInterestRate decimal.Decimal `json:"annual_interest_rate"`
	DayCountConvention string          `json:"day_count_convention"`
	StartDate          time.Time       `json:"start_date"`
	DueDate            *time.Time      `json:"due_date,omitempty"`
	Status             Status          `json:"status"`
	Version            int             `json:"version"`
	Notes              *string         `json:"notes,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

type LedgerEvent struct {
	ID        uuid.UUID       `json:"id"`
	UserID    uuid.UUID       `json:"user_id"`
	LoanID    *uuid.UUID      `json:"loan_id,omitempty"`
	EventType string          `json:"event_type"`
	ActorType string          `json:"actor_type"`
	ActorID   *uuid.UUID      `json:"actor_id,omitempty"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}
