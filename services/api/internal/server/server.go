package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Server struct {
	httpServer *http.Server
}

func NewServer(port string, handler http.Handler) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%s", port),
			Handler:      handler,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}
}

func (s *Server) Start() error {
	shutdownErr := make(chan error, 1)

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		slog.Info("shutdown signal received, initiating graceful shutdown")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		shutdownErr <- s.httpServer.Shutdown(ctx)
	}()

	slog.Info("http server listening", "addr", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("http server failure: %w", err)
	}

	err := <-shutdownErr
	if err != nil {
		return fmt.Errorf("failed to shutdown cleanly: %w", err)
	}

	slog.Info("http server stopped cleanly")
	return nil
}

