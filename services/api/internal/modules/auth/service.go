package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	internalErrors "tavrocred/services/api/internal/errors"
)

type Service interface {
	Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error)
	Login(ctx context.Context, req LoginRequest) (*AuthResponse, error)
	Refresh(ctx context.Context, req RefreshRequest) (*AuthResponse, error)
	Logout(ctx context.Context, req LogoutRequest) error
	GetMe(ctx context.Context, userID uuid.UUID) (*UserResponse, error)
}

type authService struct {
	repo         Repository
	hasher       *PasswordHasher
	tokenManager *TokenManager
}

func NewService(repo Repository, hasher *PasswordHasher, tokenManager *TokenManager) Service {
	return &authService{
		repo:         repo,
		hasher:       hasher,
		tokenManager: tokenManager,
	}
}

func (s *authService) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	if err := req.SanitizeAndValidate(); err != nil {
		return nil, err
	}

	hash, err := s.hasher.Hash(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user, err := s.repo.CreateUser(ctx, req.Email, hash, req.FullName)
	if err != nil {
		return nil, err
	}

	return s.generateTokenPair(ctx, user)
}

func (s *authService) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	if err := req.SanitizeAndValidate(); err != nil {
		return nil, err
	}

	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, internalErrors.ErrNotFound) {
			return nil, internalErrors.NewUnauthorizedError("invalid email or password")
		}
		return nil, err
	}

	match, err := s.hasher.Verify(req.Password, user.PasswordHash)
	if err != nil || !match {
		return nil, internalErrors.NewUnauthorizedError("invalid email or password")
	}

	return s.generateTokenPair(ctx, user)
}

func (s *authService) Refresh(ctx context.Context, req RefreshRequest) (*AuthResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	tokenHash := HashRefreshToken(req.RefreshToken)
	existingToken, err := s.repo.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, internalErrors.ErrNotFound) {
			return nil, internalErrors.NewUnauthorizedError("invalid or expired refresh token")
		}
		return nil, err
	}

	if !existingToken.IsActive() {
		return nil, internalErrors.NewUnauthorizedError("refresh token has been revoked or expired")
	}

	// Revoke the old refresh token (token rotation)
	if err := s.repo.RevokeRefreshToken(ctx, tokenHash); err != nil {
		return nil, fmt.Errorf("failed to revoke old refresh token: %w", err)
	}

	user, err := s.repo.GetUserByID(ctx, existingToken.UserID)
	if err != nil {
		return nil, err
	}

	return s.generateTokenPair(ctx, user)
}

func (s *authService) Logout(ctx context.Context, req LogoutRequest) error {
	if err := req.Validate(); err != nil {
		return err
	}

	tokenHash := HashRefreshToken(req.RefreshToken)
	return s.repo.RevokeRefreshToken(ctx, tokenHash)
}

func (s *authService) GetMe(ctx context.Context, userID uuid.UUID) (*UserResponse, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	res := ToUserResponse(user)
	return &res, nil
}

func (s *authService) generateTokenPair(ctx context.Context, user *User) (*AuthResponse, error) {
	accessToken, expiresAt, err := s.tokenManager.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	rawRefreshToken, refreshHash, refreshExpiresAt, err := s.tokenManager.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	if _, err := s.repo.CreateRefreshToken(ctx, user.ID, refreshHash, refreshExpiresAt); err != nil {
		return nil, fmt.Errorf("failed to persist refresh token: %w", err)
	}

	expiresIn := int64(time.Until(expiresAt).Seconds())
	if expiresIn < 0 {
		expiresIn = 0
	}

	return &AuthResponse{
		User:         ToUserResponse(user),
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		ExpiresIn:    expiresIn,
		Tokens: TokenPair{
			AccessToken:  accessToken,
			RefreshToken: rawRefreshToken,
			ExpiresIn:    expiresIn,
		},
	}, nil
}
