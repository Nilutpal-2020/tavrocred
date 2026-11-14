package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

type DB struct {
	Pool *pgxpool.Pool
	url  string
}

func Connect(ctx context.Context, databaseURL string) (*DB, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	config.MaxConns = 25
	config.MinConns = 2
	config.MaxConnLifetime = 1 * time.Hour
	config.MaxConnIdleTime = 15 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{
		Pool: pool,
		url:  databaseURL,
	}, nil
}

func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}

func (db *DB) Ping(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

func ResolveMigrationsDir(dir string) string {
	candidates := []string{
		dir,
		"migrations",
		filepath.Join("services", "api", "migrations"),
		filepath.Join("..", "migrations"),
		filepath.Join("..", "..", "migrations"),
		filepath.Join("..", "..", "services", "api", "migrations"),
		"/app/migrations",
	}

	for _, c := range candidates {
		if c == "" {
			continue
		}
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c
		}
	}

	return dir
}

func (db *DB) RunMigrations(migrationsDir string) error {
	targetDir := ResolveMigrationsDir(migrationsDir)

	sqlDB, err := sql.Open("pgx", db.url)
	if err != nil {
		return fmt.Errorf("failed to open sql db for migrations: %w", err)
	}
	defer sqlDB.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	slog.Info("running database migrations", "dir", targetDir)
	if err := goose.Up(sqlDB, targetDir); err != nil {
		return fmt.Errorf("failed to run goose migrations: %w", err)
	}

	slog.Info("database migrations applied successfully")
	return nil
}
