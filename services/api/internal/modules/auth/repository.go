package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"tavrocred/services/api/gen"
	internalErrors "tavrocred/services/api/internal/errors"
)

type Repository interface {
	CreateUser(ctx context.Context, email, passwordHash, fullName string) (*User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	CreateRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (*RefreshToken, error)
	GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID) error
}

type pgRepository struct {
	q gen.Querier
}

func NewRepository(q gen.Querier) Repository {
	return &pgRepository{q: q}
}

func (r *pgRepository) CreateUser(ctx context.Context, email, passwordHash, fullName string) (*User, error) {
	row, err := r.q.CreateUser(ctx, gen.CreateUserParams{
		Lower:        email,
		PasswordHash: passwordHash,
		FullName:     fullName,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return nil, internalErrors.NewConflictError("a user with this email already exists")
		}
		return nil, err
	}

	return toDomainUser(row), nil
}

func (r *pgRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	row, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, internalErrors.NewNotFoundError("user not found")
		}
		return nil, err
	}

	return toDomainUser(row), nil
}

func (r *pgRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	row, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, internalErrors.NewNotFoundError("user not found")
		}
		return nil, err
	}

	return toDomainUser(row), nil
}

func (r *pgRepository) CreateRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (*RefreshToken, error) {
	row, err := r.q.CreateRefreshToken(ctx, gen.CreateRefreshTokenParams{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	return toDomainRefreshToken(row), nil
}

func (r *pgRepository) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	row, err := r.q.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, internalErrors.NewNotFoundError("refresh token not found")
		}
		return nil, err
	}

	return toDomainRefreshToken(row), nil
}

func (r *pgRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	return r.q.RevokeRefreshToken(ctx, tokenHash)
}

func (r *pgRepository) RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	return r.q.RevokeAllUserRefreshTokens(ctx, userID)
}

func toDomainUser(u gen.Users) *User {
	return &User{
		ID:           u.ID,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		FullName:     u.FullName,
		CreatedAt:    u.CreatedAt.Time,
		UpdatedAt:    u.UpdatedAt.Time,
	}
}

func toDomainRefreshToken(rt gen.RefreshTokens) *RefreshToken {
	var revokedAt *time.Time
	if rt.RevokedAt.Valid {
		t := rt.RevokedAt.Time
		revokedAt = &t
	}

	return &RefreshToken{
		ID:        rt.ID,
		UserID:    rt.UserID,
		TokenHash: rt.TokenHash,
		ExpiresAt: rt.ExpiresAt.Time,
		RevokedAt: revokedAt,
		CreatedAt: rt.CreatedAt.Time,
	}
}
