package peer

import (
	"time"

	"github.com/google/uuid"
)

type Peer struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	Email     *string   `json:"email,omitempty"`
	Phone     *string   `json:"phone,omitempty"`
	Notes     *string   `json:"notes,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

