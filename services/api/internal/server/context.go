package server

import (
	"context"

	"github.com/google/uuid"

	internalErrors "tavrocred/services/api/internal/errors"
)

type contextKey string

const (
	UserIDKey    contextKey = "user_id"
	UserEmailKey contextKey = "user_email"
)

func SetUserContext(ctx context.Context, userID uuid.UUID, email string) context.Context {
	ctx = context.WithValue(ctx, UserIDKey, userID)
	return context.WithValue(ctx, UserEmailKey, email)
}

func GetUserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	val := ctx.Value(UserIDKey)
	if val == nil {
		return uuid.Nil, internalErrors.NewUnauthorizedError("unauthenticated request context")
	}

	userID, ok := val.(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return uuid.Nil, internalErrors.NewUnauthorizedError("invalid user id in request context")
	}

	return userID, nil
}

func GetUserEmailFromContext(ctx context.Context) string {
	if email, ok := ctx.Value(UserEmailKey).(string); ok {
		return email
	}
	return ""
}

