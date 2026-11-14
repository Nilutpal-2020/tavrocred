package auth_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internalErrors "tavrocred/services/api/internal/errors"
	"tavrocred/services/api/internal/modules/auth"
)

type mockAuthRepo struct {
	users        map[string]*auth.User // keyed by lower(email)
	usersByID    map[uuid.UUID]*auth.User
	tokensByHash map[string]*auth.RefreshToken
}

func newMockAuthRepo() *mockAuthRepo {
	return &mockAuthRepo{
		users:        make(map[string]*auth.User),
		usersByID:    make(map[uuid.UUID]*auth.User),
		tokensByHash: make(map[string]*auth.RefreshToken),
	}
}

func (m *mockAuthRepo) CreateUser(ctx context.Context, email, passwordHash, fullName string) (*auth.User, error) {
	lower := strings.ToLower(email)
	if _, exists := m.users[lower]; exists {
		return nil, internalErrors.NewConflictError("a user with this email already exists")
	}

	u := &auth.User{
		ID:           uuid.New(),
		Email:        lower,
		PasswordHash: passwordHash,
		FullName:     fullName,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	m.users[lower] = u
	m.usersByID[u.ID] = u
	return u, nil
}

func (m *mockAuthRepo) GetUserByID(ctx context.Context, id uuid.UUID) (*auth.User, error) {
	if u, exists := m.usersByID[id]; exists {
		return u, nil
	}
	return nil, internalErrors.NewNotFoundError("user not found")
}

func (m *mockAuthRepo) GetUserByEmail(ctx context.Context, email string) (*auth.User, error) {
	lower := strings.ToLower(email)
	if u, exists := m.users[lower]; exists {
		return u, nil
	}
	return nil, internalErrors.NewNotFoundError("user not found")
}

func (m *mockAuthRepo) CreateRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (*auth.RefreshToken, error) {
	rt := &auth.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now().UTC(),
	}
	m.tokensByHash[tokenHash] = rt
	return rt, nil
}

func (m *mockAuthRepo) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*auth.RefreshToken, error) {
	if rt, exists := m.tokensByHash[tokenHash]; exists {
		return rt, nil
	}
	return nil, internalErrors.NewNotFoundError("refresh token not found")
}

func (m *mockAuthRepo) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	if rt, exists := m.tokensByHash[tokenHash]; exists {
		now := time.Now().UTC()
		rt.RevokedAt = &now
	}
	return nil
}

func (m *mockAuthRepo) RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	now := time.Now().UTC()
	for _, rt := range m.tokensByHash {
		if rt.UserID == userID {
			rt.RevokedAt = &now
		}
	}
	return nil
}

func setupAuthService() (auth.Service, *mockAuthRepo) {
	repo := newMockAuthRepo()
	hasher := &auth.PasswordHasher{
		Memory:      16 * 1024,
		Iterations:  2,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	}
	tm := auth.NewTokenManager("test-jwt-secret-for-service-testing-32-bytes!", 15*time.Minute, 30*24*time.Hour)
	svc := auth.NewService(repo, hasher, tm)
	return svc, repo
}

func TestAuthService_Register(t *testing.T) {
	svc, _ := setupAuthService()
	ctx := context.Background()

	res, err := svc.Register(ctx, auth.RegisterRequest{
		Email:    "test@example.com",
		Password: "password123",
		FullName: "Test User",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, res.AccessToken)
	assert.NotEmpty(t, res.RefreshToken)
	assert.Equal(t, "test@example.com", res.User.Email)
	assert.Equal(t, "Test User", res.User.FullName)

	// Duplicate register should fail
	_, err = svc.Register(ctx, auth.RegisterRequest{
		Email:    "test@example.com",
		Password: "password123",
		FullName: "Test User 2",
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, internalErrors.ErrConflict)
}

func TestAuthService_Login(t *testing.T) {
	svc, _ := setupAuthService()
	ctx := context.Background()

	_, err := svc.Register(ctx, auth.RegisterRequest{
		Email:    "login@example.com",
		Password: "correctpassword",
		FullName: "Login User",
	})
	require.NoError(t, err)

	// Successful login
	loginRes, err := svc.Login(ctx, auth.LoginRequest{
		Email:    "login@example.com",
		Password: "correctpassword",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, loginRes.AccessToken)
	assert.NotEmpty(t, loginRes.RefreshToken)

	// Wrong password
	_, err = svc.Login(ctx, auth.LoginRequest{
		Email:    "login@example.com",
		Password: "wrongpassword",
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, internalErrors.ErrUnauthorized)

	// Non-existent user
	_, err = svc.Login(ctx, auth.LoginRequest{
		Email:    "unknown@example.com",
		Password: "correctpassword",
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, internalErrors.ErrUnauthorized)
}

func TestAuthService_Refresh(t *testing.T) {
	svc, _ := setupAuthService()
	ctx := context.Background()

	regRes, err := svc.Register(ctx, auth.RegisterRequest{
		Email:    "refresh@example.com",
		Password: "password123",
		FullName: "Refresh User",
	})
	require.NoError(t, err)

	// Valid refresh rotates token
	refRes, err := svc.Refresh(ctx, auth.RefreshRequest{
		RefreshToken: regRes.RefreshToken,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, refRes.AccessToken)
	assert.NotEmpty(t, refRes.RefreshToken)
	assert.NotEqual(t, regRes.RefreshToken, refRes.RefreshToken)

	// Using the old refresh token must fail because it was revoked during rotation
	_, err = svc.Refresh(ctx, auth.RefreshRequest{
		RefreshToken: regRes.RefreshToken,
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, internalErrors.ErrUnauthorized)
}

func TestAuthService_Logout(t *testing.T) {
	svc, _ := setupAuthService()
	ctx := context.Background()

	regRes, err := svc.Register(ctx, auth.RegisterRequest{
		Email:    "logout@example.com",
		Password: "password123",
		FullName: "Logout User",
	})
	require.NoError(t, err)

	// Logout
	err = svc.Logout(ctx, auth.LogoutRequest{
		RefreshToken: regRes.RefreshToken,
	})
	require.NoError(t, err)

	// After logout, refresh must fail
	_, err = svc.Refresh(ctx, auth.RefreshRequest{
		RefreshToken: regRes.RefreshToken,
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, internalErrors.ErrUnauthorized)
}

func TestAuthService_GetMe(t *testing.T) {
	svc, _ := setupAuthService()
	ctx := context.Background()

	regRes, err := svc.Register(ctx, auth.RegisterRequest{
		Email:    "me@example.com",
		Password: "password123",
		FullName: "Me User",
	})
	require.NoError(t, err)

	user, err := svc.GetMe(ctx, regRes.User.ID)
	require.NoError(t, err)
	assert.Equal(t, regRes.User.ID, user.ID)
	assert.Equal(t, "me@example.com", user.Email)

	// Unknown user ID
	_, err = svc.GetMe(ctx, uuid.New())
	assert.Error(t, err)
	assert.ErrorIs(t, err, internalErrors.ErrNotFound)
}

