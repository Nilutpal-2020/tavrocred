package auth_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tavrocred/services/api/internal/modules/auth"
)

func TestPasswordHasher_HashAndVerify(t *testing.T) {
	hasher := &auth.PasswordHasher{
		Memory:      16 * 1024,
		Iterations:  2,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	}

	password := "SecureP@ssw0rd123!"

	hash, err := hasher.Hash(password)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.True(t, strings.HasPrefix(hash, "$argon2id$v=19$"))

	// Verify with correct password
	match, err := hasher.Verify(password, hash)
	require.NoError(t, err)
	assert.True(t, match)

	// Verify with wrong password
	match, err = hasher.Verify("WrongPassword456!", hash)
	require.NoError(t, err)
	assert.False(t, match)
}

func TestPasswordHasher_DifferentHashesForSamePassword(t *testing.T) {
	hasher := &auth.PasswordHasher{
		Memory:      16 * 1024,
		Iterations:  2,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	}

	password := "SamePassword123!"

	hash1, err := hasher.Hash(password)
	require.NoError(t, err)

	hash2, err := hasher.Hash(password)
	require.NoError(t, err)

	assert.NotEqual(t, hash1, hash2, "salts must differ between runs")

	match1, _ := hasher.Verify(password, hash1)
	match2, _ := hasher.Verify(password, hash2)
	assert.True(t, match1)
	assert.True(t, match2)
}

func TestPasswordHasher_InvalidHashFormat(t *testing.T) {
	hasher := auth.DefaultPasswordHasher()

	_, err := hasher.Verify("password", "invalid-hash-string")
	assert.ErrorIs(t, err, auth.ErrInvalidHash)

	_, err = hasher.Verify("password", "$bcrypt$v=19$m=65536,t=3,p=2$salt$hash")
	assert.ErrorIs(t, err, auth.ErrInvalidHash)
}
