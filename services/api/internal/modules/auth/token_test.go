package auth_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tavrocred/services/api/internal/modules/auth"
)

func TestTokenManager_AccessTokens(t *testing.T) {
	secret := "test-secret-key-that-is-sufficiently-long-32-chars!"
	tm := auth.NewTokenManager(secret, 15*time.Minute, 30*24*time.Hour)

	userID := uuid.New()
	email := "alice@example.com"

	tokenStr, expiresAt, err := tm.GenerateAccessToken(userID, email)
	require.NoError(t, err)
	assert.NotEmpty(t, tokenStr)
	assert.True(t, expiresAt.After(time.Now()))

	claims, err := tm.ValidateAccessToken(tokenStr)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, "tavrocred", claims.Issuer)
}

func TestTokenManager_ExpiredAccessToken(t *testing.T) {
	secret := "test-secret-key-that-is-sufficiently-long-32-chars!"
	// Expire immediately
	tm := auth.NewTokenManager(secret, -1*time.Minute, 30*24*time.Hour)

	userID := uuid.New()
	tokenStr, _, err := tm.GenerateAccessToken(userID, "bob@example.com")
	require.NoError(t, err)

	_, err = tm.ValidateAccessToken(tokenStr)
	assert.ErrorIs(t, err, auth.ErrTokenExpired)
}

func TestTokenManager_TamperedAccessToken(t *testing.T) {
	secret := "test-secret-key-that-is-sufficiently-long-32-chars!"
	tm := auth.NewTokenManager(secret, 15*time.Minute, 30*24*time.Hour)

	tokenStr, _, err := tm.GenerateAccessToken(uuid.New(), "carol@example.com")
	require.NoError(t, err)

	// Tamper with the token
	tampered := tokenStr + "tampered"
	_, err = tm.ValidateAccessToken(tampered)
	assert.Error(t, err)
	assert.ErrorIs(t, err, auth.ErrTokenMalformed)

	// Validate with different secret
	otherTm := auth.NewTokenManager("different-secret-key-that-is-32-chars-long!", 15*time.Minute, 30*24*time.Hour)
	_, err = otherTm.ValidateAccessToken(tokenStr)
	assert.Error(t, err)
}

func TestTokenManager_RefreshTokens(t *testing.T) {
	tm := auth.NewTokenManager("secret", 15*time.Minute, 7*24*time.Hour)

	token1, hash1, exp1, err := tm.GenerateRefreshToken()
	require.NoError(t, err)
	assert.NotEmpty(t, token1)
	assert.NotEmpty(t, hash1)
	assert.Len(t, hash1, 64) // SHA-256 hex is 64 characters
	assert.True(t, exp1.After(time.Now()))

	// Hashes must be deterministic
	assert.Equal(t, hash1, auth.HashRefreshToken(token1))

	token2, hash2, _, err := tm.GenerateRefreshToken()
	require.NoError(t, err)
	assert.NotEqual(t, token1, token2)
	assert.NotEqual(t, hash1, hash2)
}

