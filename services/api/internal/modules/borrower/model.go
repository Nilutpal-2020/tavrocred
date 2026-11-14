package borrower

import (
	"time"

	"github.com/google/uuid"
)

type TokenPurpose string

const (
	PurposePortalAccess TokenPurpose = "PORTAL_ACCESS"
)

type AcknowledgementStatus string

const (
	StatusPending      AcknowledgementStatus = "PENDING"
	StatusAcknowledged AcknowledgementStatus = "ACKNOWLEDGED"
	StatusDisputed     AcknowledgementStatus = "DISPUTED"
)

type BorrowerAccessToken struct {
	ID        uuid.UUID    `json:"id"`
	LoanID    uuid.UUID    `json:"loan_id"`
	PeerID    uuid.UUID    `json:"peer_id"`
	UserID    uuid.UUID    `json:"user_id"`
	TokenHash string       `json:"-"`
	Purpose   TokenPurpose `json:"purpose"`
	ExpiresAt time.Time    `json:"expires_at"`
	RevokedAt *time.Time   `json:"revoked_at,omitempty"`
	UsedAt    *time.Time   `json:"used_at,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
}

type LoanAcknowledgement struct {
	ID            uuid.UUID             `json:"id"`
	LoanID        uuid.UUID             `json:"loan_id"`
	PeerID        uuid.UUID             `json:"peer_id"`
	Version       int                   `json:"version"`
	Status        AcknowledgementStatus `json:"status"`
	DisputeReason *string               `json:"dispute_reason,omitempty"`
	IPAddress     *string               `json:"ip_address,omitempty"`
	UserAgent     *string               `json:"user_agent,omitempty"`
	RespondedAt   time.Time             `json:"responded_at"`
	CreatedAt     time.Time             `json:"created_at"`
}

type BorrowerClaims struct {
	LoanID uuid.UUID `json:"loan_id"`
	PeerID uuid.UUID `json:"peer_id"`
	UserID uuid.UUID `json:"user_id"`
	Role   string    `json:"role"`
}
