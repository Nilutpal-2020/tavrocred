package borrower

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"tavrocred/services/api/gen"
	internalErrors "tavrocred/services/api/internal/errors"
	"tavrocred/services/api/internal/modules/loan"
)

type Repository interface {
	CreateAccessToken(ctx context.Context, t *BorrowerAccessToken) (*BorrowerAccessToken, error)
	GetActiveAccessTokenByHash(ctx context.Context, tokenHash string) (*BorrowerAccessToken, error)
	RevokeAccessTokensByLoan(ctx context.Context, loanID, userID uuid.UUID) error
	MarkAccessTokenUsed(ctx context.Context, id uuid.UUID) error

	CreateAcknowledgement(ctx context.Context, ack *LoanAcknowledgement) (*LoanAcknowledgement, error)
	GetLatestAcknowledgementByLoan(ctx context.Context, loanID uuid.UUID) (*LoanAcknowledgement, error)
	ListAcknowledgementsByLoan(ctx context.Context, loanID uuid.UUID) ([]*LoanAcknowledgement, error)

	GetLoanByIDAndPeer(ctx context.Context, loanID, peerID uuid.UUID) (*loan.Loan, error)
	UpdateLoanStatusByPeer(ctx context.Context, loanID, peerID uuid.UUID, status loan.Status) (*loan.Loan, error)
}

type pgRepository struct {
	q gen.Querier
}

func NewRepository(q gen.Querier) Repository {
	return &pgRepository{q: q}
}

func (r *pgRepository) CreateAccessToken(ctx context.Context, t *BorrowerAccessToken) (*BorrowerAccessToken, error) {
	row, err := r.q.CreateBorrowerAccessToken(ctx, gen.CreateBorrowerAccessTokenParams{
		LoanID:    t.LoanID,
		PeerID:    t.PeerID,
		UserID:    t.UserID,
		TokenHash: t.TokenHash,
		Purpose:   string(t.Purpose),
		ExpiresAt: pgtype.Timestamptz{Time: t.ExpiresAt, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	return toDomainToken(row), nil
}

func (r *pgRepository) GetActiveAccessTokenByHash(ctx context.Context, tokenHash string) (*BorrowerAccessToken, error) {
	row, err := r.q.GetActiveBorrowerAccessTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, internalErrors.NewNotFoundError("valid active borrower access token not found")
		}
		return nil, err
	}

	return toDomainToken(row), nil
}

func (r *pgRepository) RevokeAccessTokensByLoan(ctx context.Context, loanID, userID uuid.UUID) error {
	_, err := r.q.RevokeBorrowerAccessTokensByLoan(ctx, gen.RevokeBorrowerAccessTokensByLoanParams{
		LoanID: loanID,
		UserID: userID,
	})
	return err
}

func (r *pgRepository) MarkAccessTokenUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.q.MarkBorrowerAccessTokenUsed(ctx, id)
	return err
}

func (r *pgRepository) CreateAcknowledgement(ctx context.Context, ack *LoanAcknowledgement) (*LoanAcknowledgement, error) {
	var dispReasonPg pgtype.Text
	if ack.DisputeReason != nil {
		dispReasonPg = pgtype.Text{String: *ack.DisputeReason, Valid: true}
	}
	var ipPg pgtype.Text
	if ack.IPAddress != nil {
		ipPg = pgtype.Text{String: *ack.IPAddress, Valid: true}
	}
	var uaPg pgtype.Text
	if ack.UserAgent != nil {
		uaPg = pgtype.Text{String: *ack.UserAgent, Valid: true}
	}

	row, err := r.q.CreateLoanAcknowledgement(ctx, gen.CreateLoanAcknowledgementParams{
		LoanID:        ack.LoanID,
		PeerID:        ack.PeerID,
		Version:       int32(ack.Version),
		Status:        string(ack.Status),
		DisputeReason: dispReasonPg,
		IpAddress:     ipPg,
		UserAgent:     uaPg,
		RespondedAt:   pgtype.Timestamptz{Time: ack.RespondedAt, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	return toDomainAck(row), nil
}

func (r *pgRepository) GetLatestAcknowledgementByLoan(ctx context.Context, loanID uuid.UUID) (*LoanAcknowledgement, error) {
	row, err := r.q.GetLatestAcknowledgementByLoan(ctx, loanID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, internalErrors.NewNotFoundError("no acknowledgement found")
		}
		return nil, err
	}

	return toDomainAck(row), nil
}

func (r *pgRepository) ListAcknowledgementsByLoan(ctx context.Context, loanID uuid.UUID) ([]*LoanAcknowledgement, error) {
	rows, err := r.q.ListAcknowledgementsByLoan(ctx, loanID)
	if err != nil {
		return nil, err
	}

	acks := make([]*LoanAcknowledgement, len(rows))
	for i, row := range rows {
		acks[i] = toDomainAck(row)
	}

	return acks, nil
}

func (r *pgRepository) GetLoanByIDAndPeer(ctx context.Context, loanID, peerID uuid.UUID) (*loan.Loan, error) {
	row, err := r.q.GetLoanByIDAndPeer(ctx, gen.GetLoanByIDAndPeerParams{
		ID:     loanID,
		PeerID: peerID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, internalErrors.NewNotFoundError("loan not found for this borrower")
		}
		return nil, err
	}

	return toDomainLoan(row), nil
}

func (r *pgRepository) UpdateLoanStatusByPeer(ctx context.Context, loanID, peerID uuid.UUID, status loan.Status) (*loan.Loan, error) {
	row, err := r.q.UpdateLoanStatusByPeer(ctx, gen.UpdateLoanStatusByPeerParams{
		ID:     loanID,
		PeerID: peerID,
		Status: string(status),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, internalErrors.NewNotFoundError("loan not found")
		}
		return nil, err
	}

	return toDomainLoan(row), nil
}

func toDomainToken(row gen.BorrowerAccessTokens) *BorrowerAccessToken {
	var revAt *time.Time
	if row.RevokedAt.Valid {
		revAt = &row.RevokedAt.Time
	}
	var usedAt *time.Time
	if row.UsedAt.Valid {
		usedAt = &row.UsedAt.Time
	}

	return &BorrowerAccessToken{
		ID:        row.ID,
		LoanID:    row.LoanID,
		PeerID:    row.PeerID,
		UserID:    row.UserID,
		TokenHash: row.TokenHash,
		Purpose:   TokenPurpose(row.Purpose),
		ExpiresAt: row.ExpiresAt.Time,
		RevokedAt: revAt,
		UsedAt:    usedAt,
		CreatedAt: row.CreatedAt.Time,
	}
}

func toDomainAck(row gen.LoanAcknowledgements) *LoanAcknowledgement {
	var dispReason *string
	if row.DisputeReason.Valid {
		dispReason = &row.DisputeReason.String
	}
	var ip *string
	if row.IpAddress.Valid {
		ip = &row.IpAddress.String
	}
	var ua *string
	if row.UserAgent.Valid {
		ua = &row.UserAgent.String
	}

	return &LoanAcknowledgement{
		ID:            row.ID,
		LoanID:        row.LoanID,
		PeerID:        row.PeerID,
		Version:       int(row.Version),
		Status:        AcknowledgementStatus(row.Status),
		DisputeReason: dispReason,
		IPAddress:     ip,
		UserAgent:     ua,
		RespondedAt:   row.RespondedAt.Time,
		CreatedAt:     row.CreatedAt.Time,
	}
}

func fromPgNumeric(num pgtype.Numeric) decimal.Decimal {
	if !num.Valid {
		return decimal.Zero
	}
	if v, err := num.Value(); err == nil && v != nil {
		if str, ok := v.(string); ok {
			d, _ := decimal.NewFromString(str)
			return d
		}
	}
	return decimal.Zero
}

func fromPgDate(d pgtype.Date) *time.Time {
	if !d.Valid {
		return nil
	}
	t := d.Time
	return &t
}

func fromPgText(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	s := t.String
	return &s
}

func toDomainLoan(row gen.Loans) *loan.Loan {
	return &loan.Loan{
		ID:                 row.ID,
		UserID:             row.UserID,
		PeerID:             row.PeerID,
		PrincipalAmount:    fromPgNumeric(row.PrincipalAmount),
		Currency:           row.Currency,
		InterestType:       loan.InterestType(row.InterestType),
		AnnualInterestRate: fromPgNumeric(row.AnnualInterestRate),
		DayCountConvention: row.DayCountConvention,
		StartDate:          row.StartDate.Time,
		DueDate:            fromPgDate(row.DueDate),
		Status:             loan.Status(row.Status),
		Version:            int(row.Version),
		Notes:              fromPgText(row.Notes),
		CreatedAt:          row.CreatedAt.Time,
		UpdatedAt:          row.UpdatedAt.Time,
	}
}
