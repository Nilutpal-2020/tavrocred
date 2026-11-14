package peer

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
	r.Route("/peers", func(r chi.Router) {
		r.Use(authMiddleware)

		r.Post("/", h.Create)
		r.Get("/", h.List)
		r.Get("/{peerID}", h.GetByID)
		r.Put("/{peerID}", h.Update)
		r.Delete("/{peerID}", h.Delete)
	})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, err := server.GetUserIDFromContext(r.Context())
	if err != nil {
		server.HandleError(w, err)
		return
	}

	var req CreatePeerRequest
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

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, err := server.GetUserIDFromContext(r.Context())
	if err != nil {
		server.HandleError(w, err)
		return
	}

	query := ListPeersQuery{
		Page:     1,
		PageSize: 20,
		Search:   r.URL.Query().Get("q"),
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

	res, meta, err := h.service.List(r.Context(), userID, query)
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

	peerIDStr := chi.URLParam(r, "peerID")
	peerID, err := uuid.Parse(peerIDStr)
	if err != nil {
		server.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid peer id", nil)
		return
	}

	res, err := h.service.GetByID(r.Context(), peerID, userID)
	if err != nil {
		server.HandleError(w, err)
		return
	}

	server.JSON(w, http.StatusOK, res)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID, err := server.GetUserIDFromContext(r.Context())
	if err != nil {
		server.HandleError(w, err)
		return
	}

	peerIDStr := chi.URLParam(r, "peerID")
	peerID, err := uuid.Parse(peerIDStr)
	if err != nil {
		server.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid peer id", nil)
		return
	}

	var req UpdatePeerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		server.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "malformed json request body", nil)
		return
	}

	res, err := h.service.Update(r.Context(), peerID, userID, req)
	if err != nil {
		server.HandleError(w, err)
		return
	}

	server.JSON(w, http.StatusOK, res)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, err := server.GetUserIDFromContext(r.Context())
	if err != nil {
		server.HandleError(w, err)
		return
	}

	peerIDStr := chi.URLParam(r, "peerID")
	peerID, err := uuid.Parse(peerIDStr)
	if err != nil {
		server.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid peer id", nil)
		return
	}

	if err := h.service.Delete(r.Context(), peerID, userID); err != nil {
		server.HandleError(w, err)
		return
	}

	server.JSON(w, http.StatusOK, map[string]string{
		"message": "peer deleted successfully",
	})
}

