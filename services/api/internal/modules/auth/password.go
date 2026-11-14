package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidHash         = errors.New("the encoded password hash is not in the correct format")
	ErrIncompatibleVersion = errors.New("incompatible version of argon2")
)

type PasswordHasher struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

func DefaultPasswordHasher() *PasswordHasher {
	return &PasswordHasher{
		Memory:      64 * 1024, // 64 MB
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
}

func (h *PasswordHasher) Hash(password string) (string, error) {
	salt := make([]byte, h.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate random salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, h.Iterations, h.Memory, h.Parallelism, h.KeyLength)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, h.Memory, h.Iterations, h.Parallelism, b64Salt, b64Hash)

	return encoded, nil
}

func (h *PasswordHasher) Verify(password, encodedHash string) (bool, error) {
	vals := strings.Split(encodedHash, "$")
	if len(vals) != 6 {
		return false, ErrInvalidHash
	}

	if vals[1] != "argon2id" {
		return false, ErrInvalidHash
	}

	var version int
	if _, err := fmt.Sscanf(vals[2], "v=%d", &version); err != nil {
		return false, ErrInvalidHash
	}
	if version != argon2.Version {
		return false, ErrIncompatibleVersion
	}

	var memory, iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(vals[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return false, ErrInvalidHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(vals[4])
	if err != nil {
		return false, ErrInvalidHash
	}

	hash, err := base64.RawStdEncoding.DecodeString(vals[5])
	if err != nil {
		return false, ErrInvalidHash
	}

	otherHash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(hash)))

	if subtle.ConstantTimeCompare(hash, otherHash) == 1 {
		return true, nil
	}

	return false, nil
}

