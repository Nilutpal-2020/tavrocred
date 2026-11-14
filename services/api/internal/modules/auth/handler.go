package auth

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"tavrocred/services/api/internal/server"
)

type UserIDGetter func(r *http.Request) (interface{}, error)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
		r.Post("/refresh", h.Refresh)
		r.Post("/logout", h.Logout)

		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Get("/me", h.GetMe)
		})
	})
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		server.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "malformed json request body", nil)
		return
	}

	res, err := h.service.Register(r.Context(), req)
	if err != nil {
		server.HandleError(w, err)
		return
	}

	server.JSON(w, http.StatusCreated, res)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		server.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "malformed json request body", nil)
		return
	}

	res, err := h.service.Login(r.Context(), req)
	if err != nil {
		server.HandleError(w, err)
		return
	}

	server.JSON(w, http.StatusOK, res)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		server.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "malformed json request body", nil)
		return
	}

	res, err := h.service.Refresh(r.Context(), req)
	if err != nil {
		server.HandleError(w, err)
		return
	}

	server.JSON(w, http.StatusOK, res)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req LogoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		server.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "malformed json request body", nil)
		return
	}

	if err := h.service.Logout(r.Context(), req); err != nil {
		server.HandleError(w, err)
		return
	}

	server.JSON(w, http.StatusOK, map[string]string{
		"message": "logged out successfully",
	})
}

func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, err := server.GetUserIDFromContext(r.Context())
	if err != nil {
		server.HandleError(w, err)
		return
	}

	res, err := h.service.GetMe(r.Context(), userID)
	if err != nil {
		server.HandleError(w, err)
		return
	}

	server.JSON(w, http.StatusOK, res)
}
