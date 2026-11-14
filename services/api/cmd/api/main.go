package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"tavrocred/services/api/gen"
	"tavrocred/services/api/internal/config"
	"tavrocred/services/api/internal/database"
	"tavrocred/services/api/internal/middleware"
	"tavrocred/services/api/internal/modules/auth"
	"tavrocred/services/api/internal/modules/peer"
	"tavrocred/services/api/internal/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("starting tavrocred api server")

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if os.Getenv("RUN_MIGRATIONS") == "true" {
		migrationsDir := os.Getenv("MIGRATIONS_DIR")
		if migrationsDir == "" {
			migrationsDir = filepath.Join("migrations")
		}
		if err := db.RunMigrations(migrationsDir); err != nil {
			slog.Error("failed to run database migrations", "error", err)
			os.Exit(1)
		}
	}

	queries := gen.New(db.Pool)

	// Auth dependencies
	hasher := &auth.PasswordHasher{
		Memory:      cfg.Argon2Memory,
		Iterations:  cfg.Argon2Iterations,
		Parallelism: cfg.Argon2Parallelism,
		SaltLength:  cfg.Argon2SaltLength,
		KeyLength:   cfg.Argon2KeyLength,
	}
	tokenManager := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTExpiration, cfg.RefreshTokenExpiration)
	authRepo := auth.NewRepository(queries)
	authService := auth.NewService(authRepo, hasher, tokenManager)
	authHandler := auth.NewHandler(authService)

	// Peer dependencies
	peerRepo := peer.NewRepository(queries)
	peerService := peer.NewService(peerRepo)
	peerHandler := peer.NewHandler(peerService)

	authMiddleware := middleware.Authenticate(tokenManager)

	router := server.NewRouter(server.RouterConfig{
		DB:             db,
		AuthHandler:    authHandler,
		PeerHandler:    peerHandler,
		AuthMiddleware: authMiddleware,
		RequestIDMW:    middleware.RequestID,
		LoggerMW:       middleware.Logger,
		RecoverMW:      middleware.Recover,
		TimeoutMW:      middleware.Timeout(30 * time.Second),
	})

	srv := server.NewServer(cfg.Port, router)
	if err := srv.Start(); err != nil {
		slog.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}

