package peer_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internalErrors "tavrocred/services/api/internal/errors"
	"tavrocred/services/api/internal/modules/peer"
)

type mockPeerRepo struct {
	peers map[uuid.UUID]*peer.Peer
}

func newMockPeerRepo() *mockPeerRepo {
	return &mockPeerRepo{
		peers: make(map[uuid.UUID]*peer.Peer),
	}
}

func (m *mockPeerRepo) Create(ctx context.Context, userID uuid.UUID, name string, email, phone, notes *string) (*peer.Peer, error) {
	p := &peer.Peer{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      name,
		Email:     email,
		Phone:     phone,
		Notes:     notes,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	m.peers[p.ID] = p
	return p, nil
}

func (m *mockPeerRepo) GetByIDAndUser(ctx context.Context, id, userID uuid.UUID) (*peer.Peer, error) {
	p, exists := m.peers[id]
	if !exists || p.UserID != userID {
		return nil, internalErrors.NewNotFoundError("peer not found")
	}
	return p, nil
}

func (m *mockPeerRepo) ListByUser(ctx context.Context, userID uuid.UUID, search string, limit, offset int32) ([]*peer.Peer, int64, error) {
	var filtered []*peer.Peer
	for _, p := range m.peers {
		if p.UserID != userID {
			continue
		}
		if search != "" {
			matchName := strings.Contains(strings.ToLower(p.Name), strings.ToLower(search))
			matchEmail := p.Email != nil && strings.Contains(strings.ToLower(*p.Email), strings.ToLower(search))
			if !matchName && !matchEmail {
				continue
			}
		}
		filtered = append(filtered, p)
	}

	total := int64(len(filtered))
	start := int(offset)
	if start > len(filtered) {
		return []*peer.Peer{}, total, nil
	}
	end := start + int(limit)
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end], total, nil
}

func (m *mockPeerRepo) Update(ctx context.Context, id, userID uuid.UUID, name string, email, phone, notes *string) (*peer.Peer, error) {
	p, exists := m.peers[id]
	if !exists || p.UserID != userID {
		return nil, internalErrors.NewNotFoundError("peer not found")
	}

	p.Name = name
	p.Email = email
	p.Phone = phone
	p.Notes = notes
	p.UpdatedAt = time.Now().UTC()

	return p, nil
}

func (m *mockPeerRepo) Delete(ctx context.Context, id, userID uuid.UUID) error {
	p, exists := m.peers[id]
	if !exists || p.UserID != userID {
		return internalErrors.NewNotFoundError("peer not found")
	}
	delete(m.peers, id)
	return nil
}

func setupPeerService() (peer.Service, *mockPeerRepo) {
	repo := newMockPeerRepo()
	svc := peer.NewService(repo)
	return svc, repo
}

func TestPeerService_CreateAndGet(t *testing.T) {
	svc, _ := setupPeerService()
	ctx := context.Background()

	userA := uuid.New()
	userB := uuid.New()

	email := "borrower@example.com"
	res, err := svc.Create(ctx, userA, peer.CreatePeerRequest{
		Name:  "Borrower One",
		Email: &email,
	})
	require.NoError(t, err)
	assert.Equal(t, "Borrower One", res.Name)
	assert.Equal(t, userA, res.UserID)

	// User A can access their own peer
	got, err := svc.GetByID(ctx, res.ID, userA)
	require.NoError(t, err)
	assert.Equal(t, res.ID, got.ID)

	// User B cannot access User A's peer (Strict Owner Scoping)
	_, err = svc.GetByID(ctx, res.ID, userB)
	assert.Error(t, err)
	assert.ErrorIs(t, err, internalErrors.ErrNotFound)
}

func TestPeerService_List(t *testing.T) {
	svc, _ := setupPeerService()
	ctx := context.Background()

	userA := uuid.New()
	userB := uuid.New()

	// Create 3 peers for User A
	for i := 1; i <= 3; i++ {
		_, err := svc.Create(ctx, userA, peer.CreatePeerRequest{
			Name: "Peer A " + string(rune('0'+i)),
		})
		require.NoError(t, err)
	}

	// Create 1 peer for User B
	_, err := svc.Create(ctx, userB, peer.CreatePeerRequest{
		Name: "Peer B",
	})
	require.NoError(t, err)

	// List for User A should only see 3 peers
	peersA, metaA, err := svc.List(ctx, userA, peer.ListPeersQuery{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Len(t, peersA, 3)
	assert.Equal(t, int64(3), metaA.TotalItems)
	assert.Equal(t, 1, metaA.TotalPages)

	// List for User B should only see 1 peer
	peersB, metaB, err := svc.List(ctx, userB, peer.ListPeersQuery{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Len(t, peersB, 1)
	assert.Equal(t, int64(1), metaB.TotalItems)
}

func TestPeerService_Update(t *testing.T) {
	svc, _ := setupPeerService()
	ctx := context.Background()

	userA := uuid.New()
	userB := uuid.New()

	created, err := svc.Create(ctx, userA, peer.CreatePeerRequest{
		Name: "Initial Name",
	})
	require.NoError(t, err)

	// User B cannot update User A's peer
	_, err = svc.Update(ctx, created.ID, userB, peer.UpdatePeerRequest{
		Name: "Hacked Name",
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, internalErrors.ErrNotFound)

	// User A can update their peer
	updated, err := svc.Update(ctx, created.ID, userA, peer.UpdatePeerRequest{
		Name: "Updated Name",
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.Name)
}

func TestPeerService_Delete(t *testing.T) {
	svc, _ := setupPeerService()
	ctx := context.Background()

	userA := uuid.New()
	userB := uuid.New()

	created, err := svc.Create(ctx, userA, peer.CreatePeerRequest{
		Name: "To Delete",
	})
	require.NoError(t, err)

	// User B cannot delete User A's peer
	err = svc.Delete(ctx, created.ID, userB)
	assert.Error(t, err)
	assert.ErrorIs(t, err, internalErrors.ErrNotFound)

	// User A can delete their peer
	err = svc.Delete(ctx, created.ID, userA)
	require.NoError(t, err)

	// Once deleted, retrieval fails
	_, err = svc.GetByID(ctx, created.ID, userA)
	assert.Error(t, err)
	assert.ErrorIs(t, err, internalErrors.ErrNotFound)
}

