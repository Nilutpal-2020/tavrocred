package repayment

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"tavrocred/services/api/internal/server"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	// Nested routes under /loans/{loanID}/repayments
	r.Route("/loans/{loanID}/repayments", func(r chi.Router) {
		r.Use(authMiddleware)

		r.Post("/", h.Record)
		r.Get("/", h.ListByLoan)
	})

	// Direct routes under /repayments
	r.Route("/repayments", func(r chi.Router) {
		r.Use(authMiddleware)

		r.Get("/{repaymentID}", h.GetByID)
		r.Post("/{repaymentID}/confirm", h.Confirm)
		r.Post("/{repaymentID}/reject", h.Reject)
		r.Post("/{repaymentID}/reverse", h.Reverse)
	})
}

func (h *Handler) Record(w http.ResponseWriter, r *http.Request) {
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

	var req RecordRepaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		server.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "malformed json request body", nil)
		return
	}

	res, err := h.service.Record(r.Context(), loanID, userID, req)
	if err != nil {
		server.HandleError(w, err)
		return
	}

	server.JSON(w, http.StatusCreated, res)
}

func (h *Handler) ListByLoan(w http.ResponseWriter, r *http.Request) {
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

	query := ListRepaymentsQuery{
		Page:     1,
		PageSize: 20,
		Status:   r.URL.Query().Get("status"),
	}

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			query.Page = p
		}
	}

	if pageSizeStr := r.URL.Query().Get("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
			query.PageSize = ps
		}
	}

	res, meta, err := h.service.ListByLoan(r.Context(), loanID, userID, query)
	if err != nil {
		server.HandleError(w, err)
		return
	}

	server.JSONWithMeta(w, http.StatusOK, res, meta)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	userID, err := server.GetUserIDFromContext(r.Context())
	if err != nil {
		server.HandleError(w, err)
		return
	}

	repaymentIDStr := chi.URLParam(r, "repaymentID")
	repaymentID, err := uuid.Parse(repaymentIDStr)
	if err != nil {
		server.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid repayment id format", nil)
		return
	}

	res, err := h.service.GetByID(r.Context(), repaymentID, userID)
	if err != nil {
		server.HandleError(w, err)
		return
	}

	server.JSON(w, http.StatusOK, res)
}

func (h *Handler) Confirm(w http.ResponseWriter, r *http.Request) {
	userID, err := server.GetUserIDFromContext(r.Context())
	if err != nil {
		server.HandleError(w, err)
		return
	}

	repaymentIDStr := chi.URLParam(r, "repaymentID")
	repaymentID, err := uuid.Parse(repaymentIDStr)
	if err != nil {
		server.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid repayment id format", nil)
		return
	}

	res, err := h.service.Confirm(r.Context(), repaymentID, userID)
	if err != nil {
		server.HandleError(w, err)
		return
	}

	server.JSON(w, http.StatusOK, res)
}

func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	userID, err := server.GetUserIDFromContext(r.Context())
	if err != nil {
		server.HandleError(w, err)
		return
	}

	repaymentIDStr := chi.URLParam(r, "repaymentID")
	repaymentID, err := uuid.Parse(repaymentIDStr)
	if err != nil {
		server.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid repayment id format", nil)
		return
	}

	var req RejectRepaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		server.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "malformed json request body", nil)
		return
	}

	res, err := h.service.Reject(r.Context(), repaymentID, userID, req)
	if err != nil {
		server.HandleError(w, err)
		return
	}

	server.JSON(w, http.StatusOK, res)
}

func (h *Handler) Reverse(w http.ResponseWriter, r *http.Request) {
	userID, err := server.GetUserIDFromContext(r.Context())
	if err != nil {
		server.HandleError(w, err)
		return
	}

	repaymentIDStr := chi.URLParam(r, "repaymentID")
	repaymentID, err := uuid.Parse(repaymentIDStr)
	if err != nil {
		server.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid repayment id format", nil)
		return
	}

	var req ReverseRepaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		server.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "malformed json request body", nil)
		return
	}

	res, err := h.service.Reverse(r.Context(), repaymentID, userID, req)
	if err != nil {
		server.HandleError(w, err)
		return
	}

	server.JSON(w, http.StatusOK, res)
}
