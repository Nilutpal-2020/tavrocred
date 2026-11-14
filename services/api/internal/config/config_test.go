package config_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tavrocred/services/api/internal/config"
)

func TestConfig_LoadDefaults(t *testing.T) {
	// Clear relevant env vars
	os.Unsetenv("PORT")
	os.Unsetenv("ENV")
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("JWT_SECRET")

	t.Setenv("DATABASE_URL", "postgres://localhost/test")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "development", cfg.Env)
	assert.Equal(t, "postgres://localhost/test", cfg.DatabaseURL)
	assert.NotEmpty(t, cfg.JWTSecret)
	assert.Equal(t, uint32(64*1024), cfg.Argon2Memory)
}

func TestConfig_Validate_MissingRequired(t *testing.T) {
	cfg := &config.Config{
		Port:        "",
		DatabaseURL: "",
		JWTSecret:   "",
	}
	assert.Error(t, cfg.Validate())

	cfg.Port = "8080"
	assert.Error(t, cfg.Validate())

	cfg.DatabaseURL = "postgres://localhost/test"
	assert.Error(t, cfg.Validate())

	cfg.JWTSecret = "short"
	assert.NoError(t, cfg.Validate()) // In non-production, short is allowed
}

func TestConfig_Validate_ProductionJWTLength(t *testing.T) {
	cfg := &config.Config{
		Port:        "8080",
		Env:         "production",
		DatabaseURL: "postgres://localhost/test",
		JWTSecret:   "short-secret",
	}

	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least 32 characters in production")

	cfg.JWTSecret = "this-is-a-valid-production-secret-with-more-than-32-chars"
	assert.NoError(t, cfg.Validate())
}
