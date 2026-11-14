package peer

import (
	"strings"
	"time"

	"github.com/google/uuid"
	internalErrors "tavrocred/services/api/internal/errors"
)

type CreatePeerRequest struct {
	Name  string  `json:"name"`
	Email *string `json:"email,omitempty"`
	Phone *string `json:"phone,omitempty"`
	Notes *string `json:"notes,omitempty"`
}

func (r *CreatePeerRequest) SanitizeAndValidate() error {
	r.Name = strings.TrimSpace(r.Name)
	details := make(map[string]string)

	if r.Name == "" {
		details["name"] = "name is required"
	} else if len(r.Name) > 255 {
		details["name"] = "name cannot exceed 255 characters"
	}

	if r.Email != nil {
		clean := strings.TrimSpace(strings.ToLower(*r.Email))
		if clean == "" {
			r.Email = nil
		} else {
			if !strings.Contains(clean, "@") || !strings.Contains(clean, ".") {
				details["email"] = "valid email address is required"
			}
			r.Email = &clean
		}
	}

	if r.Phone != nil {
		clean := strings.TrimSpace(*r.Phone)
		if clean == "" {
			r.Phone = nil
		} else {
			if len(clean) > 50 {
				details["phone"] = "phone number cannot exceed 50 characters"
			}
			r.Phone = &clean
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
		return internalErrors.NewValidationError("invalid peer data", details)
	}

	return nil
}

type UpdatePeerRequest struct {
	Name  string  `json:"name"`
	Email *string `json:"email,omitempty"`
	Phone *string `json:"phone,omitempty"`
	Notes *string `json:"notes,omitempty"`
}

func (r *UpdatePeerRequest) SanitizeAndValidate() error {
	r.Name = strings.TrimSpace(r.Name)
	details := make(map[string]string)

	if r.Name == "" {
		details["name"] = "name is required"
	} else if len(r.Name) > 255 {
		details["name"] = "name cannot exceed 255 characters"
	}

	if r.Email != nil {
		clean := strings.TrimSpace(strings.ToLower(*r.Email))
		if clean == "" {
			r.Email = nil
		} else {
			if !strings.Contains(clean, "@") || !strings.Contains(clean, ".") {
				details["email"] = "valid email address is required"
			}
			r.Email = &clean
		}
	}

	if r.Phone != nil {
		clean := strings.TrimSpace(*r.Phone)
		if clean == "" {
			r.Phone = nil
		} else {
			if len(clean) > 50 {
				details["phone"] = "phone number cannot exceed 50 characters"
			}
			r.Phone = &clean
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
		return internalErrors.NewValidationError("invalid peer data", details)
	}

	return nil
}

type PeerResponse struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	Email     *string   `json:"email,omitempty"`
	Phone     *string   `json:"phone,omitempty"`
	Notes     *string   `json:"notes,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func ToPeerResponse(p *Peer) PeerResponse {
	return PeerResponse{
		ID:        p.ID,
		UserID:    p.UserID,
		Name:      p.Name,
		Email:     p.Email,
		Phone:     p.Phone,
		Notes:     p.Notes,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

type ListPeersQuery struct {
	Page     int
	PageSize int
	Search   string
}

func (q *ListPeersQuery) Sanitize() {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = 20
	} else if q.PageSize > 100 {
		q.PageSize = 100
	}
	q.Search = strings.TrimSpace(q.Search)
}

func (q *ListPeersQuery) Offset() int32 {
	return int32((q.Page - 1) * q.PageSize)
}

func (q *ListPeersQuery) Limit() int32 {
	return int32(q.PageSize)
}
