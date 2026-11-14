package peer

import (
	"context"
	"math"

	"github.com/google/uuid"

	"tavrocred/services/api/internal/server"
)

type Service interface {
	Create(ctx context.Context, userID uuid.UUID, req CreatePeerRequest) (*PeerResponse, error)
	GetByID(ctx context.Context, id, userID uuid.UUID) (*PeerResponse, error)
	List(ctx context.Context, userID uuid.UUID, query ListPeersQuery) ([]PeerResponse, *server.Meta, error)
	Update(ctx context.Context, id, userID uuid.UUID, req UpdatePeerRequest) (*PeerResponse, error)
	Delete(ctx context.Context, id, userID uuid.UUID) error
}

type peerService struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &peerService{repo: repo}
}

func (s *peerService) Create(ctx context.Context, userID uuid.UUID, req CreatePeerRequest) (*PeerResponse, error) {
	if err := req.SanitizeAndValidate(); err != nil {
		return nil, err
	}

	peer, err := s.repo.Create(ctx, userID, req.Name, req.Email, req.Phone, req.Notes)
	if err != nil {
		return nil, err
	}

	res := ToPeerResponse(peer)
	return &res, nil
}

func (s *peerService) GetByID(ctx context.Context, id, userID uuid.UUID) (*PeerResponse, error) {
	peer, err := s.repo.GetByIDAndUser(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	res := ToPeerResponse(peer)
	return &res, nil
}

func (s *peerService) List(ctx context.Context, userID uuid.UUID, query ListPeersQuery) ([]PeerResponse, *server.Meta, error) {
	query.Sanitize()

	peers, total, err := s.repo.ListByUser(ctx, userID, query.Search, query.Limit(), query.Offset())
	if err != nil {
		return nil, nil, err
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(query.PageSize)))
	}

	meta := &server.Meta{
		Page:       query.Page,
		PageSize:   query.PageSize,
		TotalItems: total,
		TotalPages: totalPages,
	}

	responses := make([]PeerResponse, len(peers))
	for i, p := range peers {
		responses[i] = ToPeerResponse(p)
	}

	return responses, meta, nil
}

func (s *peerService) Update(ctx context.Context, id, userID uuid.UUID, req UpdatePeerRequest) (*PeerResponse, error) {
	if err := req.SanitizeAndValidate(); err != nil {
		return nil, err
	}

	peer, err := s.repo.Update(ctx, id, userID, req.Name, req.Email, req.Phone, req.Notes)
	if err != nil {
		return nil, err
	}

	res := ToPeerResponse(peer)
	return &res, nil
}

func (s *peerService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return s.repo.Delete(ctx, id, userID)
}

