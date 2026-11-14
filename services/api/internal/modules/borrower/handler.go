package borrower

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	internalErrors "tavrocred/services/api/internal/errors"
	"tavrocred/services/api/internal/server"
)

type contextKey string

const (
	borrowerClaimsContextKey contextKey = "borrowerClaims"
)

func SetBorrowerClaimsInContext(ctx context.Context, claims *BorrowerClaims) context.Context {
	return context.WithValue(ctx, borrowerClaimsContextKey, claims)
}

func GetBorrowerClaimsFromContext(ctx context.Context) (*BorrowerClaims, error) {
	claims, ok := ctx.Value(borrowerClaimsContextKey).(*BorrowerClaims)
	if !ok || claims == nil {
		return nil, internalErrors.NewUnauthorizedError("borrower authentication required")
	}
	return claims, nil
}

type Handler struct {
	service      Service
	tokenManager TokenManager
}

func NewHandler(service Service, tokenManager TokenManager) *Handler {
	return &Handler{
		service:      service,
		tokenManager: tokenManager,
	}
}

func (h *Handler) RegisterRoutes(r chi.Router, lenderAuthMiddleware func(http.Handler) http.Handler) {
	// Lender management routes (authenticated)
	r.Route("/loans/{loanID}/borrower-link", func(r chi.Router) {
		r.Use(lenderAuthMiddleware)

		r.Post("/", h.GenerateBorrowerLink)
		r.Post("/revoke", h.RevokeBorrowerLink)
	})

	// Public borrower portal routes
	r.Route("/public/borrower", func(r chi.Router) {
		// Public token exchange endpoint
		r.Post("/access", h.ExchangeToken)

		// Borrower session-authenticated endpoints
		r.Group(func(r chi.Router) {
			r.Use(h.BorrowerAuthMiddleware)

			r.Get("/loan", h.GetLoanDetails)
			r.Post("/acknowledge", h.AcknowledgeLoan)
			r.Post("/dispute", h.DisputeLoan)
			r.Post("/repayment-reported", h.ReportRepayment)
		})
	})
}

func (h *Handler) BorrowerAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			server.HandleError(w, internalErrors.NewUnauthorizedError("missing authorization header"))
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			server.HandleError(w, internalErrors.NewUnauthorizedError("invalid authorization header format"))
			return
		}

		claims, err := h.tokenManager.VerifySessionToken(parts[1])
		if err != nil {
			server.HandleError(w, err)
			return
		}

		ctx := SetBorrowerClaimsInContext(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *Handler) GenerateBorrowerLink(w http.ResponseWriter, r *http.Request) {
	userID, err := server.GetUserIDFromContext(r.Context())
	if err != nil {
		server.HandleError(w, err)
		return
	}

	loanIDStr := chi.URLParam(r, "loanID")
	loanID, err := uuid.Parse(loanIDStr)
	if err != nil {
		server.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid loan id format", nil)
		return
	}

	var req CreateBorrowerLinkRequest
	if r.Body != nil && r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	expiresIn := req.Sanitize()
	res, err := h.service.GenerateBorrowerLink(r.Context(), loanID, userID, expiresIn)
	if err != nil {
		server.HandleError(w, err)
		return
	}

	server.JSON(w, http.StatusCreated, res)
}

func (h *Handler) RevokeBorrowerLink(w http.ResponseWriter, r *http.Request) {
	userID, err := server.GetUserIDFromContext(r.Context())
	if err != nil {
		server.HandleError(w, err)
		return
	}

	loanIDStr := chi.URLParam(r, "loanID")
	loanID, err := uuid.Parse(loanIDStr)
	if err != nil {
		server.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid loan id format", nil)
		return
	}

	if err := h.service.RevokeBorrowerLink(r.Context(), loanID, userID); err != nil {
		server.HandleError(w, err)
		return
	}

	server.JSON(w, http.StatusOK, map[string]string{
		"message": "borrower access links revoked successfully",
	})
}

func (h *Handler) ExchangeToken(w http.ResponseWriter, r *http.Request) {
	var req BorrowerAccessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		server.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "malformed json request body", nil)
		return
	}

	if err := req.Validate(); err != nil {
		server.HandleError(w, err)
		return
	}

	res, err := h.service.ExchangeTokenForSession(r.Context(), req.Token)
	if err != nil {
		server.HandleError(w, err)
		return
	}

	server.JSON(w, http.StatusOK, res)
}

func (h *Handler) GetLoanDetails(w http.ResponseWriter, r *http.Request) {
	claims, err := GetBorrowerClaimsFromContext(r.Context())
	if err != nil {
		server.HandleError(w, err)
		return
	}

	res, err := h.service.GetLoanDetailsForBorrower(r.Context(), claims.LoanID, claims.PeerID)
	if err != nil {
		server.HandleError(w, err)
		return
	}

	server.JSON(w, http.StatusOK, res)
}

func (h *Handler) AcknowledgeLoan(w http.ResponseWriter, r *http.Request) {
	claims, err := GetBorrowerClaimsFromContext(r.Context())
	if err != nil {
		server.HandleError(w, err)
		return
	}

	ip, ua := extractClientMetadata(r)
	res, err := h.service.AcknowledgeLoan(r.Context(), claims.LoanID, claims.PeerID, ip, ua)
	if err != nil {
		server.HandleError(w, err)
		return
	}

	server.JSON(w, http.StatusOK, res)
}

func (h *Handler) DisputeLoan(w http.ResponseWriter, r *http.Request) {
	claims, err := GetBorrowerClaimsFromContext(r.Context())
	if err != nil {
		server.HandleError(w, err)
		return
	}

	var req BorrowerDisputeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		server.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "malformed json request body", nil)
		return
	}

	if err := req.Validate(); err != nil {
		server.HandleError(w, err)
		return
	}

	ip, ua := extractClientMetadata(r)
	res, err := h.service.DisputeLoan(r.Context(), claims.LoanID, claims.PeerID, req.Reason, ip, ua)
	if err != nil {
		server.HandleError(w, err)
		return
	}

	server.JSON(w, http.StatusOK, res)
}

func (h *Handler) ReportRepayment(w http.ResponseWriter, r *http.Request) {
	claims, err := GetBorrowerClaimsFromContext(r.Context())
	if err != nil {
		server.HandleError(w, err)
		return
	}

	var req BorrowerReportRepaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		server.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "malformed json request body", nil)
		return
	}

	res, err := h.service.ReportRepayment(r.Context(), claims.LoanID, claims.PeerID, req)
	if err != nil {
		server.HandleError(w, err)
		return
	}

	server.JSON(w, http.StatusCreated, res)
}

func extractClientMetadata(r *http.Request) (string, string) {
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.RemoteAddr
	} else {
		parts := strings.Split(ip, ",")
		ip = strings.TrimSpace(parts[0])
	}
	ua := r.UserAgent()
	return ip, ua
}
