package auth

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"tavrocred/services/api/internal/errors"
)

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

func (r *RegisterRequest) SanitizeAndValidate() error {
	r.Email = strings.TrimSpace(strings.ToLower(r.Email))
	r.FullName = strings.TrimSpace(r.FullName)

	details := make(map[string]string)
	if r.Email == "" {
		details["email"] = "email is required"
	} else if !strings.Contains(r.Email, "@") || !strings.Contains(r.Email, ".") {
		details["email"] = "valid email address is required"
	}

	if len(r.Password) < 8 {
		details["password"] = "password must be at least 8 characters"
	}

	if r.FullName == "" {
		details["full_name"] = "full name is required"
	}

	if len(details) > 0 {
		return errors.NewValidationError("invalid registration input", details)
	}

	return nil
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r *LoginRequest) SanitizeAndValidate() error {
	r.Email = strings.TrimSpace(strings.ToLower(r.Email))

	details := make(map[string]string)
	if r.Email == "" {
		details["email"] = "email is required"
	}
	if r.Password == "" {
		details["password"] = "password is required"
	}

	if len(details) > 0 {
		return errors.NewValidationError("invalid login input", details)
	}

	return nil
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (r *RefreshRequest) Validate() error {
	r.RefreshToken = strings.TrimSpace(r.RefreshToken)
	if r.RefreshToken == "" {
		return errors.NewValidationError("refresh_token is required", map[string]string{
			"refresh_token": "refresh_token cannot be blank",
		})
	}
	return nil
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (r *LogoutRequest) Validate() error {
	r.RefreshToken = strings.TrimSpace(r.RefreshToken)
	if r.RefreshToken == "" {
		return errors.NewValidationError("refresh_token is required", map[string]string{
			"refresh_token": "refresh_token cannot be blank",
		})
	}
	return nil
}

type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func ToUserResponse(u *User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		FullName:  u.FullName,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

type AuthResponse struct {
	User         UserResponse `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int64        `json:"expires_in"` // seconds
}

