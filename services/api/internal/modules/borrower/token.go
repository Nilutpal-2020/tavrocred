package borrower

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	internalErrors "tavrocred/services/api/internal/errors"
)

type TokenManager interface {
	GenerateRawToken() (rawToken string, tokenHash string, err error)
	HashToken(rawToken string) string
	SignSessionToken(loanID, peerID, userID uuid.UUID, duration time.Duration) (string, error)
	VerifySessionToken(tokenString string) (*BorrowerClaims, error)
}

type jwtTokenManager struct {
	secretKey []byte
}

func NewTokenManager(jwtSecret string) TokenManager {
	return &jwtTokenManager{
		secretKey: []byte(jwtSecret),
	}
}

func (m *jwtTokenManager) GenerateRawToken() (string, string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("failed to generate random token: %w", err)
	}

	rawToken := hex.EncodeToString(b)
	hash := m.HashToken(rawToken)
	return rawToken, hash, nil
}

func (m *jwtTokenManager) HashToken(rawToken string) string {
	h := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(h[:])
}

type borrowerJWTClaims struct {
	LoanID string `json:"loan_id"`
	PeerID string `json:"peer_id"`
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func (m *jwtTokenManager) SignSessionToken(loanID, peerID, userID uuid.UUID, duration time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := borrowerJWTClaims{
		LoanID: loanID.String(),
		PeerID: peerID.String(),
		UserID: userID.String(),
		Role:   "borrower",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "tavrocred-borrower",
			Subject:   peerID.String(),
			Audience:  jwt.ClaimStrings{"tavrocred-borrower-portal"},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign borrower session token: %w", err)
	}

	return signed, nil
}

func (m *jwtTokenManager) VerifySessionToken(tokenString string) (*BorrowerClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &borrowerJWTClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secretKey, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, internalErrors.NewUnauthorizedError("borrower session has expired")
		}
		return nil, internalErrors.NewUnauthorizedError("invalid borrower session token")
	}

	claims, ok := token.Claims.(*borrowerJWTClaims)
	if !ok || !token.Valid || claims.Role != "borrower" || claims.Issuer != "tavrocred-borrower" {
		return nil, internalErrors.NewUnauthorizedError("invalid borrower token claims")
	}

	loanID, err := uuid.Parse(claims.LoanID)
	if err != nil {
		return nil, internalErrors.NewUnauthorizedError("malformed loan ID in token")
	}
	peerID, err := uuid.Parse(claims.PeerID)
	if err != nil {
		return nil, internalErrors.NewUnauthorizedError("malformed peer ID in token")
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, internalErrors.NewUnauthorizedError("malformed user ID in token")
	}

	return &BorrowerClaims{
		LoanID: loanID,
		PeerID: peerID,
		UserID: userID,
		Role:   claims.Role,
	}, nil
}
