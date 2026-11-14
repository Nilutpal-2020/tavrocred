package peer_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tavrocred/services/api/internal/middleware"
	"tavrocred/services/api/internal/modules/auth"
	"tavrocred/services/api/internal/modules/peer"
	"tavrocred/services/api/internal/server"
)

func setupPeerTestApp() (http.Handler, *auth.TokenManager) {
	repo := newMockPeerRepo()
	peerSvc := peer.NewService(repo)
	peerHandler := peer.NewHandler(peerSvc)

	tm := auth.NewTokenManager("test-secret-for-peer-handlers-at-least-32-chars!", 15*time.Minute, 30*24*time.Hour)
	authMiddleware := middleware.Authenticate(tm)

	router := server.NewRouter(server.RouterConfig{
		PeerHandler:    peerHandler,
		AuthMiddleware: authMiddleware,
		RequestIDMW:    middleware.RequestID,
		LoggerMW:       middleware.Logger,
		RecoverMW:      middleware.Recover,
	})

	return router, tm
}

func TestPeerHandler_FullCRUDAndOwnershipIsolation(t *testing.T) {
	app, tm := setupPeerTestApp()

	userA := uuid.New()
	tokenA, _, err := tm.GenerateAccessToken(userA, "usera@example.com")
	require.NoError(t, err)

	userB := uuid.New()
	tokenB, _, err := tm.GenerateAccessToken(userB, "userb@example.com")
	require.NoError(t, err)

	// 1. Create peer as User A
	createPayload := map[string]string{
		"name":  "Charlie Borrower",
		"email": "charlie@example.com",
		"phone": "+919876543210",
		"notes": "College roommate",
	}
	body, _ := json.Marshal(createPayload)
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/peers", bytes.NewReader(body))
	reqCreate.Header.Set("Authorization", "Bearer "+tokenA)
	reqCreate.Header.Set("Content-Type", "application/json")
	recCreate := httptest.NewRecorder()

	app.ServeHTTP(recCreate, reqCreate)
	assert.Equal(t, http.StatusCreated, recCreate.Code)

	var createResp struct {
		Success bool              `json:"success"`
		Data    peer.PeerResponse `json:"data"`
	}
	err = json.Unmarshal(recCreate.Body.Bytes(), &createResp)
	require.NoError(t, err)
	peerID := createResp.Data.ID
	assert.Equal(t, "Charlie Borrower", createResp.Data.Name)

	// 2. User A can retrieve peer
	reqGetA := httptest.NewRequest(http.MethodGet, "/api/v1/peers/"+peerID.String(), nil)
	reqGetA.Header.Set("Authorization", "Bearer "+tokenA)
	recGetA := httptest.NewRecorder()
	app.ServeHTTP(recGetA, reqGetA)
	assert.Equal(t, http.StatusOK, recGetA.Code)

	// 3. User B CANNOT retrieve User A's peer (Strict Owner Scoping -> 404 Not Found)
	reqGetB := httptest.NewRequest(http.MethodGet, "/api/v1/peers/"+peerID.String(), nil)
	reqGetB.Header.Set("Authorization", "Bearer "+tokenB)
	recGetB := httptest.NewRecorder()
	app.ServeHTTP(recGetB, reqGetB)
	assert.Equal(t, http.StatusNotFound, recGetB.Code)

	// 4. User B CANNOT update User A's peer
	updatePayload := map[string]string{"name": "Malicious Update"}
	upBody, _ := json.Marshal(updatePayload)
	reqUpB := httptest.NewRequest(http.MethodPut, "/api/v1/peers/"+peerID.String(), bytes.NewReader(upBody))
	reqUpB.Header.Set("Authorization", "Bearer "+tokenB)
	reqUpB.Header.Set("Content-Type", "application/json")
	recUpB := httptest.NewRecorder()
	app.ServeHTTP(recUpB, reqUpB)
	assert.Equal(t, http.StatusNotFound, recUpB.Code)

	// 5. User A can update their peer
	validUpdate := map[string]string{"name": "Charlie Updated"}
	valUpBody, _ := json.Marshal(validUpdate)
	reqUpA := httptest.NewRequest(http.MethodPut, "/api/v1/peers/"+peerID.String(), bytes.NewReader(valUpBody))
	reqUpA.Header.Set("Authorization", "Bearer "+tokenA)
	reqUpA.Header.Set("Content-Type", "application/json")
	recUpA := httptest.NewRecorder()
	app.ServeHTTP(recUpA, reqUpA)
	assert.Equal(t, http.StatusOK, recUpA.Code)

	// 6. User B's list should not include User A's peer
	reqListB := httptest.NewRequest(http.MethodGet, "/api/v1/peers", nil)
	reqListB.Header.Set("Authorization", "Bearer "+tokenB)
	recListB := httptest.NewRecorder()
	app.ServeHTTP(recListB, reqListB)
	assert.Equal(t, http.StatusOK, recListB.Code)

	var listBResp struct {
		Success bool                `json:"success"`
		Data    []peer.PeerResponse `json:"data"`
		Meta    server.Meta         `json:"meta"`
	}
	err = json.Unmarshal(recListB.Body.Bytes(), &listBResp)
	require.NoError(t, err)
	assert.Empty(t, listBResp.Data)
	assert.Equal(t, int64(0), listBResp.Meta.TotalItems)

	// 7. User A's list should include the peer
	reqListA := httptest.NewRequest(http.MethodGet, "/api/v1/peers?page=1&page_size=10", nil)
	reqListA.Header.Set("Authorization", "Bearer "+tokenA)
	recListA := httptest.NewRecorder()
	app.ServeHTTP(recListA, reqListA)
	assert.Equal(t, http.StatusOK, recListA.Code)

	var listAResp struct {
		Success bool                `json:"success"`
		Data    []peer.PeerResponse `json:"data"`
		Meta    server.Meta         `json:"meta"`
	}
	err = json.Unmarshal(recListA.Body.Bytes(), &listAResp)
	require.NoError(t, err)
	assert.Len(t, listAResp.Data, 1)
	assert.Equal(t, int64(1), listAResp.Meta.TotalItems)

	// 8. User B CANNOT delete User A's peer
	reqDelB := httptest.NewRequest(http.MethodDelete, "/api/v1/peers/"+peerID.String(), nil)
	reqDelB.Header.Set("Authorization", "Bearer "+tokenB)
	recDelB := httptest.NewRecorder()
	app.ServeHTTP(recDelB, reqDelB)
	assert.Equal(t, http.StatusNotFound, recDelB.Code)

	// 9. User A can delete their peer
	reqDelA := httptest.NewRequest(http.MethodDelete, "/api/v1/peers/"+peerID.String(), nil)
	reqDelA.Header.Set("Authorization", "Bearer "+tokenA)
	recDelA := httptest.NewRecorder()
	app.ServeHTTP(recDelA, reqDelA)
	assert.Equal(t, http.StatusOK, recDelA.Code)

	// 10. After deletion, User A gets 404
	reqGetDeleted := httptest.NewRequest(http.MethodGet, "/api/v1/peers/"+peerID.String(), nil)
	reqGetDeleted.Header.Set("Authorization", "Bearer "+tokenA)
	recGetDeleted := httptest.NewRecorder()
	app.ServeHTTP(recGetDeleted, reqGetDeleted)
	assert.Equal(t, http.StatusNotFound, recGetDeleted.Code)
}
