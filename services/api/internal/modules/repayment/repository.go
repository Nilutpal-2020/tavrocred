package repayment

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
)

type Repository interface {
	Create(ctx context.Context, r *Repayment) (*Repayment, error)
	GetByIDAndUser(ctx context.Context, id, userID uuid.UUID) (*Repayment, error)
	ListByLoan(ctx context.Context, loanID, userID uuid.UUID, status string, limit, offset int32) ([]*Repayment, int64, error)
	GetConfirmedTotalByLoan(ctx context.Context, loanID, userID uuid.UUID) (decimal.Decimal, error)
	Confirm(ctx context.Context, id, userID, confirmedBy uuid.UUID) (*Repayment, error)
	Reject(ctx context.Context, id, userID uuid.UUID, reason string) (*Repayment, error)
	Reverse(ctx context.Context, id, userID uuid.UUID, reason string) (*Repayment, error)
}

type pgRepository struct {
	q gen.Querier
}

func NewRepository(q gen.Querier) Repository {
	return &pgRepository{q: q}
}

func (r *pgRepository) Create(ctx context.Context, rep *Repayment) (*Repayment, error) {
	var confAtPg pgtype.Timestamptz
	if rep.ConfirmedAt != nil {
		confAtPg = pgtype.Timestamptz{Time: *rep.ConfirmedAt, Valid: true}
	}
	var confByPg pgtype.UUID
	if rep.ConfirmedBy != nil {
		confByPg = pgtype.UUID{Bytes: *rep.ConfirmedBy, Valid: true}
	}

	row, err := r.q.CreateRepayment(ctx, gen.CreateRepaymentParams{
		UserID:            rep.UserID,
		LoanID:            rep.LoanID,
		Amount:            toPgNumeric(rep.Amount),
		Currency:          rep.Currency,
		PaymentDate:       toPgDate(rep.PaymentDate),
		PaymentMethod:     rep.PaymentMethod,
		ExternalReference: toPgText(rep.ExternalReference),
		Status:            string(rep.Status),
		Source:            string(rep.Source),
		ConfirmedAt:       confAtPg,
		ConfirmedBy:       confByPg,
		Notes:             toPgText(rep.Notes),
	})
	if err != nil {
		return nil, err
	}

	return toDomainRepayment(row), nil
}

func (r *pgRepository) GetByIDAndUser(ctx context.Context, id, userID uuid.UUID) (*Repayment, error) {
	row, err := r.q.GetRepaymentByIDAndUser(ctx, gen.GetRepaymentByIDAndUserParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, internalErrors.NewNotFoundError("repayment not found")
		}
		return nil, err
	}

	return toDomainRepayment(row), nil
}

func (r *pgRepository) ListByLoan(ctx context.Context, loanID, userID uuid.UUID, status string, limit, offset int32) ([]*Repayment, int64, error) {
	total, err := r.q.CountRepaymentsByLoan(ctx, gen.CountRepaymentsByLoanParams{
		UserID:  userID,
		LoanID:  loanID,
		Column3: status,
	})
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.q.ListRepaymentsByLoan(ctx, gen.ListRepaymentsByLoanParams{
		UserID:  userID,
		LoanID:  loanID,
		Column3: status,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, 0, err
	}

	repayments := make([]*Repayment, len(rows))
	for i, row := range rows {
		repayments[i] = toDomainRepayment(row)
	}

	return repayments, total, nil
}

func (r *pgRepository) GetConfirmedTotalByLoan(ctx context.Context, loanID, userID uuid.UUID) (decimal.Decimal, error) {
	num, err := r.q.GetConfirmedRepaymentsTotalByLoan(ctx, gen.GetConfirmedRepaymentsTotalByLoanParams{
		LoanID: loanID,
		UserID: userID,
	})
	if err != nil {
		return decimal.Zero, err
	}

	return fromPgNumeric(num), nil
}

func (r *pgRepository) Confirm(ctx context.Context, id, userID, confirmedBy uuid.UUID) (*Repayment, error) {
	row, err := r.q.ConfirmRepayment(ctx, gen.ConfirmRepaymentParams{
		ID:          id,
		UserID:      userID,
		ConfirmedBy: pgtype.UUID{Bytes: confirmedBy, Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, internalErrors.NewNotFoundError("repayment not found or not in pending verification status")
		}
		return nil, err
	}

	return toDomainRepayment(row), nil
}

func (r *pgRepository) Reject(ctx context.Context, id, userID uuid.UUID, reason string) (*Repayment, error) {
	row, err := r.q.RejectRepayment(ctx, gen.RejectRepaymentParams{
		ID:              id,
		UserID:          userID,
		RejectionReason: toPgText(&reason),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, internalErrors.NewNotFoundError("repayment not found or not in pending verification status")
		}
		return nil, err
	}

	return toDomainRepayment(row), nil
}

func (r *pgRepository) Reverse(ctx context.Context, id, userID uuid.UUID, reason string) (*Repayment, error) {
	row, err := r.q.ReverseRepayment(ctx, gen.ReverseRepaymentParams{
		ID:             id,
		UserID:         userID,
		ReversalReason: toPgText(&reason),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, internalErrors.NewNotFoundError("repayment not found or not in confirmed status")
		}
		return nil, err
	}

	return toDomainRepayment(row), nil
}

func toPgDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}

func fromPgDate(d pgtype.Date) time.Time {
	return d.Time
}

func toPgNumeric(d decimal.Decimal) pgtype.Numeric {
	var num pgtype.Numeric
	_ = num.Scan(d.StringFixed(2))
	return num
}

func fromPgNumeric(num pgtype.Numeric) decimal.Decimal {
	if !num.Valid {
		return decimal.Zero
	}
	v, err := num.Value()
	if err != nil || v == nil {
		return decimal.Zero
	}
	if s, ok := v.(string); ok {
		d, err := decimal.NewFromString(s)
		if err == nil {
			return d
		}
	}
	return decimal.Zero
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

func toDomainRepayment(row gen.Repayments) *Repayment {
	var confAt *time.Time
	if row.ConfirmedAt.Valid {
		t := row.ConfirmedAt.Time
		confAt = &t
	}
	var confBy *uuid.UUID
	if row.ConfirmedBy.Valid {
		id := uuid.UUID(row.ConfirmedBy.Bytes)
		confBy = &id
	}
	var revAt *time.Time
	if row.ReversedAt.Valid {
		t := row.ReversedAt.Time
		revAt = &t
	}

	return &Repayment{
		ID:                row.ID,
		UserID:            row.UserID,
		LoanID:            row.LoanID,
		Amount:            fromPgNumeric(row.Amount),
		Currency:          row.Currency,
		PaymentDate:       fromPgDate(row.PaymentDate),
		PaymentMethod:     row.PaymentMethod,
		ExternalReference: fromPgText(row.ExternalReference),
		Status:            Status(row.Status),
		Source:            Source(row.Source),
		ConfirmedAt:       confAt,
		ConfirmedBy:       confBy,
		RejectionReason:   fromPgText(row.RejectionReason),
		ReversedAt:        revAt,
		ReversalReason:    fromPgText(row.ReversalReason),
		Notes:             fromPgText(row.Notes),
		CreatedAt:         row.CreatedAt.Time,
		UpdatedAt:         row.UpdatedAt.Time,
	}
}
