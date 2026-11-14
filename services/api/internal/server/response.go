package server

import (
	"encoding/json"
	"errors"
	"net/http"

	internalErrors "tavrocred/services/api/internal/errors"
)

type Response struct {
	Success bool  `json:"success"`
	Data    any   `json:"data,omitempty"`
	Meta    *Meta `json:"meta,omitempty"`
}

type Meta struct {
	Page       int   `json:"page,omitempty"`
	PageSize   int   `json:"page_size,omitempty"`
	TotalItems int64 `json:"total_items,omitempty"`
	TotalPages int   `json:"total_pages,omitempty"`
}

type ErrorResponse struct {
	Success bool        `json:"success"`
	Error   ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Response{
		Success: true,
		Data:    data,
	})
}

func JSONWithMeta(w http.ResponseWriter, status int, data any, meta *Meta) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Response{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

func Error(w http.ResponseWriter, status int, code, message string, details map[string]string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Success: false,
		Error: ErrorDetail{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

func HandleError(w http.ResponseWriter, err error) {
	var appErr *internalErrors.AppError
	if errors.As(err, &appErr) {
		status := http.StatusInternalServerError
		switch appErr.Code {
		case "VALIDATION_ERROR":
			status = http.StatusUnprocessableEntity
		case "RESOURCE_CONFLICT":
			status = http.StatusConflict
		case "NOT_FOUND":
			status = http.StatusNotFound
		case "UNAUTHORIZED":
			status = http.StatusUnauthorized
		case "FORBIDDEN":
			status = http.StatusForbidden
		}
		Error(w, status, appErr.Code, appErr.Message, appErr.Details)
		return
	}

	if errors.Is(err, internalErrors.ErrUnauthorized) {
		Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required", nil)
		return
	}
	if errors.Is(err, internalErrors.ErrForbidden) {
		Error(w, http.StatusForbidden, "FORBIDDEN", "access denied", nil)
		return
	}
	if errors.Is(err, internalErrors.ErrNotFound) {
		Error(w, http.StatusNotFound, "NOT_FOUND", "resource not found", nil)
		return
	}
	if errors.Is(err, internalErrors.ErrConflict) {
		Error(w, http.StatusConflict, "RESOURCE_CONFLICT", "resource conflict", nil)
		return
	}
	if errors.Is(err, internalErrors.ErrValidation) {
		Error(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "invalid request payload", nil)
		return
	}

	Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred", nil)
}

