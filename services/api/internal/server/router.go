package server

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type DatabasePinger interface {
	Ping(ctx context.Context) error
}

type RouterConfig struct {
	DB             DatabasePinger
	AuthHandler    RouteRegisterer
	PeerHandler    RouteRegisterer
	AuthMiddleware func(http.Handler) http.Handler
	RequestIDMW    func(http.Handler) http.Handler
	LoggerMW       func(http.Handler) http.Handler
	RecoverMW      func(http.Handler) http.Handler
	TimeoutMW      func(http.Handler) http.Handler
}

type RouteRegisterer interface {
	RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler)
}

func NewRouter(cfg RouterConfig) http.Handler {
	r := chi.NewRouter()

	// Global middleware stack
	if cfg.RequestIDMW != nil {
		r.Use(cfg.RequestIDMW)
	}
	if cfg.LoggerMW != nil {
		r.Use(cfg.LoggerMW)
	}
	if cfg.RecoverMW != nil {
		r.Use(cfg.RecoverMW)
	}
	if cfg.TimeoutMW != nil {
		r.Use(cfg.TimeoutMW)
	}

	// Health and readiness endpoints
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		JSON(w, http.StatusOK, map[string]any{
			"status":    "healthy",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		if cfg.DB != nil {
			if err := cfg.DB.Ping(ctx); err != nil {
				Error(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "database is not reachable", nil)
				return
			}
		}

		JSON(w, http.StatusOK, map[string]any{
			"status":    "ready",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	// API v1 routes
	r.Route("/api/v1", func(api chi.Router) {
		if cfg.AuthHandler != nil {
			cfg.AuthHandler.RegisterRoutes(api, cfg.AuthMiddleware)
		}
		if cfg.PeerHandler != nil {
			cfg.PeerHandler.RegisterRoutes(api, cfg.AuthMiddleware)
		}
	})

	return r
}

