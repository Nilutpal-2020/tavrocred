package borrower_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tavrocred/services/api/internal/modules/borrower"
)

func TestTokenManager_GenerateAndHash(t *testing.T) {
	tm := borrower.NewTokenManager("test-jwt-secret-key-at-least-32-bytes-long!")

	rawToken, hash, err := tm.GenerateRawToken()
	require.NoError(t, err)
	assert.Len(t, rawToken, 64) // 32 bytes hex encoded = 64 characters
	assert.Len(t, hash, 64)     // SHA-256 hex encoded = 64 characters

	// Verify hashing matches
	computedHash := tm.HashToken(rawToken)
	assert.Equal(t, hash, computedHash)

	// Ensure different calls produce different tokens
	rawToken2, hash2, _ := tm.GenerateRawToken()
	assert.NotEqual(t, rawToken, rawToken2)
	assert.NotEqual(t, hash, hash2)
}

func TestTokenManager_SessionTokenLifecycle(t *testing.T) {
	tm := borrower.NewTokenManager("test-jwt-secret-key-at-least-32-bytes-long!")

	loanID := uuid.New()
	peerID := uuid.New()
	userID := uuid.New()

	t.Run("sign and verify valid token", func(t *testing.T) {
		sessionToken, err := tm.SignSessionToken(loanID, peerID, userID, 1*time.Hour)
		require.NoError(t, err)
		assert.NotEmpty(t, sessionToken)

		claims, err := tm.VerifySessionToken(sessionToken)
		require.NoError(t, err)
		assert.Equal(t, loanID, claims.LoanID)
		assert.Equal(t, peerID, claims.PeerID)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, "borrower", claims.Role)
	})

	t.Run("expired session token", func(t *testing.T) {
		sessionToken, err := tm.SignSessionToken(loanID, peerID, userID, -1*time.Minute)
		require.NoError(t, err)

		_, err = tm.VerifySessionToken(sessionToken)
		require.Error(t, err)
	})

	t.Run("invalid token signature", func(t *testing.T) {
		otherTM := borrower.NewTokenManager("different-secret-key-for-testing-32-bytes!!")
		sessionToken, err := otherTM.SignSessionToken(loanID, peerID, userID, 1*time.Hour)
		require.NoError(t, err)

		_, err = tm.VerifySessionToken(sessionToken)
		require.Error(t, err)
	})
}
