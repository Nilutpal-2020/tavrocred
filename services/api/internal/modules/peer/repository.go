package peer

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"tavrocred/services/api/gen"
	internalErrors "tavrocred/services/api/internal/errors"
)

type Repository interface {
	Create(ctx context.Context, userID uuid.UUID, name string, email, phone, notes *string) (*Peer, error)
	GetByIDAndUser(ctx context.Context, id, userID uuid.UUID) (*Peer, error)
	ListByUser(ctx context.Context, userID uuid.UUID, search string, limit, offset int32) ([]*Peer, int64, error)
	Update(ctx context.Context, id, userID uuid.UUID, name string, email, phone, notes *string) (*Peer, error)
	Delete(ctx context.Context, id, userID uuid.UUID) error
}

type pgRepository struct {
	q gen.Querier
}

func NewRepository(q gen.Querier) Repository {
	return &pgRepository{q: q}
}

func (r *pgRepository) Create(ctx context.Context, userID uuid.UUID, name string, email, phone, notes *string) (*Peer, error) {
	row, err := r.q.CreatePeer(ctx, gen.CreatePeerParams{
		UserID: userID,
		Name:   name,
		Email:  toPgText(email),
		Phone:  toPgText(phone),
		Notes:  toPgText(notes),
	})
	if err != nil {
		return nil, err
	}

	return toDomainPeer(row), nil
}

func (r *pgRepository) GetByIDAndUser(ctx context.Context, id, userID uuid.UUID) (*Peer, error) {
	row, err := r.q.GetPeerByIDAndUser(ctx, gen.GetPeerByIDAndUserParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, internalErrors.NewNotFoundError("peer not found")
		}
		return nil, err
	}

	return toDomainPeer(row), nil
}

func (r *pgRepository) ListByUser(ctx context.Context, userID uuid.UUID, search string, limit, offset int32) ([]*Peer, int64, error) {
	total, err := r.q.CountPeersByUser(ctx, gen.CountPeersByUserParams{
		UserID:  userID,
		Column2: search,
	})
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.q.ListPeersByUser(ctx, gen.ListPeersByUserParams{
		UserID:  userID,
		Column2: search,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, 0, err
	}

	peers := make([]*Peer, len(rows))
	for i, row := range rows {
		peers[i] = toDomainPeer(row)
	}

	return peers, total, nil
}

func (r *pgRepository) Update(ctx context.Context, id, userID uuid.UUID, name string, email, phone, notes *string) (*Peer, error) {
	row, err := r.q.UpdatePeer(ctx, gen.UpdatePeerParams{
		ID:     id,
		UserID: userID,
		Name:   name,
		Email:  toPgText(email),
		Phone:  toPgText(phone),
		Notes:  toPgText(notes),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, internalErrors.NewNotFoundError("peer not found")
		}
		return nil, err
	}

	return toDomainPeer(row), nil
}

func (r *pgRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	rows, err := r.q.DeletePeer(ctx, gen.DeletePeerParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return internalErrors.NewNotFoundError("peer not found")
	}

	return nil
}

func toPgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func fromPgText(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	s := t.String
	return &s
}

func toDomainPeer(p gen.Peers) *Peer {
	return &Peer{
		ID:        p.ID,
		UserID:    p.UserID,
		Name:      p.Name,
		Email:     fromPgText(p.Email),
		Phone:     fromPgText(p.Phone),
		Notes:     fromPgText(p.Notes),
		CreatedAt: p.CreatedAt.Time,
		UpdatedAt: p.UpdatedAt.Time,
	}
}
