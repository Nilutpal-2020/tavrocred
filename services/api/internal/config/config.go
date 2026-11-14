package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port                   string
	Env                    string
	DatabaseURL            string
	JWTSecret              string
	JWTExpiration          time.Duration
	RefreshTokenExpiration time.Duration
	CORSAllowedOrigins     []string
	Argon2Memory           uint32
	Argon2Iterations       uint32
	Argon2Parallelism      uint8
	Argon2SaltLength       uint32
	Argon2KeyLength        uint32
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:                   getEnv("PORT", "8080"),
		Env:                    getEnv("ENV", "development"),
		DatabaseURL:            getEnv("DATABASE_URL", ""),
		JWTSecret:              getEnv("JWT_SECRET", "super-secret-tavrocred-jwt-development-key-32bytes!"),
		JWTExpiration:          15 * time.Minute,
		RefreshTokenExpiration: 30 * 24 * time.Hour,
		CORSAllowedOrigins:     strings.Split(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000"), ","),
		Argon2Memory:           64 * 1024, // 64 MB
		Argon2Iterations:       3,
		Argon2Parallelism:      2,
		Argon2SaltLength:       16,
		Argon2KeyLength:        32,
	}

	if expStr := os.Getenv("JWT_EXPIRATION_MINUTES"); expStr != "" {
		if minutes, err := strconv.Atoi(expStr); err == nil && minutes > 0 {
			cfg.JWTExpiration = time.Duration(minutes) * time.Minute
		}
	}

	if refStr := os.Getenv("REFRESH_TOKEN_EXPIRATION_DAYS"); refStr != "" {
		if days, err := strconv.Atoi(refStr); err == nil && days > 0 {
			cfg.RefreshTokenExpiration = time.Duration(days) * 24 * time.Hour
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.Port == "" {
		return errors.New("PORT is required")
	}

	if c.DatabaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}

	if c.JWTSecret == "" {
		return errors.New("JWT_SECRET is required")
	}

	if c.Env == "production" {
		if len(c.JWTSecret) < 32 {
			return errors.New("JWT_SECRET must be at least 32 characters in production")
		}
	}

	return nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
