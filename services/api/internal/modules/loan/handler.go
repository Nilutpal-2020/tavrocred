package loan

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
	r.Route("/loans", func(r chi.Router) {
		r.Use(authMiddleware)

		r.Post("/", h.Create)
		r.Get("/", h.List)
		r.Get("/{loanID}", h.GetByID)
		r.Put("/{loanID}", h.UpdateDraft)
		r.Post("/{loanID}/activate", h.Activate)
		r.Post("/{loanID}/cancel", h.Cancel)
		r.Post("/{loanID}/dispute", h.Dispute)
		r.Post("/{loanID}/amend", h.Amend)
		r.Get("/{loanID}/events", h.ListAuditEvents)
	})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, err := server.GetUserIDFromContext(r.Context())
	if err != nil {
		server.HandleError(w, err)
		return
	}

	var req CreateLoanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		server.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "malformed json request body", nil)
		return
	}

	res, err := h.service.Create(r.Context(), userID, req)
	if err != nil {
		server.HandleError(w, err)
		return
	}

	server.JSON(w, http.StatusCreated, res)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
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

	res, err := h.service.GetByID(r.Context(), loanID, userID)
	if err != nil {
		server.HandleError(w, err)
		return
	}

	server.JSON(w, http.StatusOK, res)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, err := server.GetUserIDFromContext(r.Context())
	if err != nil {
		server.HandleError(w, err)
		return
	}

	query := ListLoansQuery{
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

	if peerIDStr := r.URL.Query().Get("peer_id"); peerIDStr != "" {
		if pid, err := uuid.Parse(peerIDStr); err == nil {
			query.PeerID = &pid
		}
	}

	res, meta, err := h.service.List(r.Context(), userID, query)
	if err != nil {
		server.HandleError(w, err)
		return
	}

	server.JSONWithMeta(w, http.StatusOK, res, meta)
}

func (h *Handler) UpdateDraft(w http.ResponseWriter, r *http.Request) {
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

	var req UpdateLoanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		server.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "malformed json request body", nil)
		return
	}

	res, err := h.service.UpdateDraft(r.Context(), loanID, userID, req)
	if err != nil {
		server.HandleError(w, err)
		return
	}

	server.JSON(w, http.StatusOK, res)
}

func (h *Handler) Activate(w http.ResponseWriter, r *http.Request) {
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

	res, err := h.service.Activate(r.Context(), loanID, userID)
	if err != nil {
		server.HandleError(w, err)
		return
	}

	server.JSON(w, http.StatusOK, res)
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
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

	var req CancelLoanRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	res, err := h.service.Cancel(r.Context(), loanID, userID, req)
	if err != nil {
		server.HandleError(w, err)
		return
	}

	server.JSON(w, http.StatusOK, res)
}

func (h *Handler) Dispute(w http.ResponseWriter, r *http.Request) {
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

	var req DisputeLoanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		server.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "malformed json request body", nil)
		return
	}

	res, err := h.service.Dispute(r.Context(), loanID, userID, req)
	if err != nil {
		server.HandleError(w, err)
		return
	}

	server.JSON(w, http.StatusOK, res)
}

func (h *Handler) Amend(w http.ResponseWriter, r *http.Request) {
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

	var req AmendLoanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		server.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "malformed json request body", nil)
		return
	}

	res, err := h.service.Amend(r.Context(), loanID, userID, req)
	if err != nil {
		server.HandleError(w, err)
		return
	}

	server.JSON(w, http.StatusOK, res)
}

func (h *Handler) ListAuditEvents(w http.ResponseWriter, r *http.Request) {
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

	events, err := h.service.ListAuditEvents(r.Context(), loanID, userID)
	if err != nil {
		server.HandleError(w, err)
		return
	}

	server.JSON(w, http.StatusOK, events)
}
