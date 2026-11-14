package auth_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"tavrocred/services/api/internal/modules/auth"
)

func TestRegisterRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		req     auth.RegisterRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: auth.RegisterRequest{
				Email:    "test@example.com",
				Password: "password123",
				FullName: "John Doe",
			},
			wantErr: false,
		},
		{
			name: "missing email",
			req: auth.RegisterRequest{
				Email:    "",
				Password: "password123",
				FullName: "John Doe",
			},
			wantErr: true,
		},
		{
			name: "invalid email format",
			req: auth.RegisterRequest{
				Email:    "notanemail",
				Password: "password123",
				FullName: "John Doe",
			},
			wantErr: true,
		},
		{
			name: "short password",
			req: auth.RegisterRequest{
				Email:    "test@example.com",
				Password: "short",
				FullName: "John Doe",
			},
			wantErr: true,
		},
		{
			name: "missing full name",
			req: auth.RegisterRequest{
				Email:    "test@example.com",
				Password: "password123",
				FullName: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.SanitizeAndValidate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoginRequest_Validation(t *testing.T) {
	valid := auth.LoginRequest{
		Email:    "user@example.com",
		Password: "password",
	}
	assert.NoError(t, valid.SanitizeAndValidate())

	missingEmail := auth.LoginRequest{
		Email:    "",
		Password: "password",
	}
	assert.Error(t, missingEmail.SanitizeAndValidate())

	missingPass := auth.LoginRequest{
		Email:    "user@example.com",
		Password: "",
	}
	assert.Error(t, missingPass.SanitizeAndValidate())
}

func TestRefreshAndLogoutRequest_Validation(t *testing.T) {
	validRefresh := auth.RefreshRequest{RefreshToken: "valid-token"}
	assert.NoError(t, validRefresh.Validate())

	blankRefresh := auth.RefreshRequest{RefreshToken: "   "}
	assert.Error(t, blankRefresh.Validate())

	validLogout := auth.LogoutRequest{RefreshToken: "valid-token"}
	assert.NoError(t, validLogout.Validate())

	blankLogout := auth.LogoutRequest{RefreshToken: "   "}
	assert.Error(t, blankLogout.Validate())
}

