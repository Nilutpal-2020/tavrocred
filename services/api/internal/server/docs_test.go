package server_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tavrocred/services/api/internal/server"
)

func TestDocsEndpoints(t *testing.T) {
	router := server.NewRouter(server.RouterConfig{})

	// 1. GET /docs (Swagger UI)
	reqDocs := httptest.NewRequest(http.MethodGet, "/docs", nil)
	recDocs := httptest.NewRecorder()
	router.ServeHTTP(recDocs, reqDocs)
	assert.Equal(t, http.StatusOK, recDocs.Code)
	assert.Contains(t, recDocs.Header().Get("Content-Type"), "text/html")
	assert.Contains(t, recDocs.Body.String(), "SwaggerUIBundle")
	assert.Contains(t, recDocs.Body.String(), "/docs/openapi.yaml")

	// 2. GET /docs/ (Trailing slash)
	reqDocsSlash := httptest.NewRequest(http.MethodGet, "/docs/", nil)
	recDocsSlash := httptest.NewRecorder()
	router.ServeHTTP(recDocsSlash, reqDocsSlash)
	assert.Equal(t, http.StatusOK, recDocsSlash.Code)
	assert.Contains(t, recDocsSlash.Body.String(), "SwaggerUIBundle")

	// 3. GET /docs/openapi.yaml (Raw spec)
	reqSpec := httptest.NewRequest(http.MethodGet, "/docs/openapi.yaml", nil)
	recSpec := httptest.NewRecorder()
	router.ServeHTTP(recSpec, reqSpec)
	assert.Equal(t, http.StatusOK, recSpec.Code)
	assert.Contains(t, recSpec.Header().Get("Content-Type"), "application/yaml")
	assert.Contains(t, recSpec.Body.String(), "openapi: 3.1.0")
	assert.Contains(t, recSpec.Body.String(), "Tavrocred API")

	// 4. GET /openapi.yaml (Root alias)
	reqRootSpec := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	recRootSpec := httptest.NewRecorder()
	router.ServeHTTP(recRootSpec, reqRootSpec)
	assert.Equal(t, http.StatusOK, recRootSpec.Code)
	assert.Contains(t, recRootSpec.Header().Get("Content-Type"), "application/yaml")

	// 5. GET /docs/scalar (Scalar UI)
	reqScalar := httptest.NewRequest(http.MethodGet, "/docs/scalar", nil)
	recScalar := httptest.NewRecorder()
	router.ServeHTTP(recScalar, reqScalar)
	assert.Equal(t, http.StatusOK, recScalar.Code)
	assert.Contains(t, recScalar.Header().Get("Content-Type"), "text/html")
	assert.Contains(t, recScalar.Body.String(), "@scalar/api-reference")

	// 6. Verify embedded openapi.yaml matches contracts specification
	contractsPath := filepath.Join("..", "..", "..", "packages", "contracts", "openapi.yaml")
	if _, err := os.Stat(contractsPath); err == nil {
		contractsBytes, err := os.ReadFile(contractsPath)
		require.NoError(t, err)
		assert.Equal(t, string(contractsBytes), recSpec.Body.String(), "Embedded openapi.yaml must stay in sync with packages/contracts/openapi.yaml")
	}
}
