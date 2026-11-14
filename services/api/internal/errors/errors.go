package errors

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound           = errors.New("resource not found")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden: access denied")
	ErrConflict           = errors.New("resource already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrTokenRevoked       = errors.New("token has been revoked")
	ErrValidation         = errors.New("validation failed")
	ErrInternal           = errors.New("internal server error")
)

type AppError struct {
	Err     error
	Code    string
	Message string
	Details map[string]string
}

func (e *AppError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Err.Error()
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewValidationError(message string, details map[string]string) *AppError {
	return &AppError{
		Err:     ErrValidation,
		Code:    "VALIDATION_ERROR",
		Message: message,
		Details: details,
	}
}

func NewConflictError(message string) *AppError {
	return &AppError{
		Err:     ErrConflict,
		Code:    "RESOURCE_CONFLICT",
		Message: message,
	}
}

func NewNotFoundError(message string) *AppError {
	return &AppError{
		Err:     ErrNotFound,
		Code:    "NOT_FOUND",
		Message: message,
	}
}

func NewUnauthorizedError(message string) *AppError {
	return &AppError{
		Err:     ErrUnauthorized,
		Code:    "UNAUTHORIZED",
		Message: message,
	}
}

func NewForbiddenError(message string) *AppError {
	return &AppError{
		Err:     ErrForbidden,
		Code:    "FORBIDDEN",
		Message: message,
	}
}
