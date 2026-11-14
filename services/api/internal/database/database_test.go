package database_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tavrocred/services/api/internal/database"
)

func TestResolveMigrationsDir(t *testing.T) {
	// From services/api/internal/database, resolving any standard relative path
	// should find the migrations folder
	dir := database.ResolveMigrationsDir("migrations")
	require.NotEmpty(t, dir)
	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Resolving root relative 'services/api/migrations'
	dirFromRootRel := database.ResolveMigrationsDir(filepath.Join("services", "api", "migrations"))
	require.NotEmpty(t, dirFromRootRel)
	infoRoot, err := os.Stat(dirFromRootRel)
	require.NoError(t, err)
	assert.True(t, infoRoot.IsDir())
}
