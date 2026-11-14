package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrTokenExpired   = errors.New("token has expired")
	ErrTokenMalformed = errors.New("token is malformed")
)

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	jwt.RegisteredClaims
}

type TokenManager struct {
	secret                 []byte
	jwtExpiration          time.Duration
	refreshTokenExpiration time.Duration
}

func NewTokenManager(secret string, jwtExpiration, refreshTokenExpiration time.Duration) *TokenManager {
	return &TokenManager{
		secret:                 []byte(secret),
		jwtExpiration:          jwtExpiration,
		refreshTokenExpiration: refreshTokenExpiration,
	}
}

func (tm *TokenManager) GenerateAccessToken(userID uuid.UUID, email string) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(tm.jwtExpiration)

	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "tavrocred",
			Subject:   userID.String(),
			Audience:  jwt.ClaimStrings{"tavrocred-api"},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(tm.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign access token: %w", err)
	}

	return tokenString, expiresAt, nil
}

func (tm *TokenManager) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return tm.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("%w: %v", ErrTokenMalformed, err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenMalformed
	}

	return claims, nil
}

func (tm *TokenManager) GenerateRefreshToken() (token string, hash string, expiresAt time.Time, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", time.Time{}, fmt.Errorf("failed to generate random bytes for refresh token: %w", err)
	}

	token = base64.RawURLEncoding.EncodeToString(b)
	hash = HashRefreshToken(token)
	expiresAt = time.Now().UTC().Add(tm.refreshTokenExpiration)

	return token, hash, expiresAt, nil
}

func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
