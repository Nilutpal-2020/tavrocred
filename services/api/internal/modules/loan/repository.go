package loan

import (
	"context"
	"encoding/json"
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
	Create(ctx context.Context, l *Loan) (*Loan, error)
	GetByIDAndUser(ctx context.Context, id, userID uuid.UUID) (*Loan, error)
	ListByUser(ctx context.Context, userID uuid.UUID, peerID *uuid.UUID, status string, limit, offset int32) ([]*Loan, int64, error)
	UpdateDraft(ctx context.Context, l *Loan) (*Loan, error)
	UpdateStatus(ctx context.Context, id, userID uuid.UUID, newStatus Status) (*Loan, error)
	Amend(ctx context.Context, id, userID uuid.UUID, principal, rate decimal.Decimal, interestType InterestType, dueDate *time.Time, notes *string) (*Loan, error)
	CreateLedgerEvent(ctx context.Context, event *LedgerEvent) (*LedgerEvent, error)
	ListLedgerEvents(ctx context.Context, loanID, userID uuid.UUID) ([]*LedgerEvent, error)
}

type pgRepository struct {
	q gen.Querier
}

func NewRepository(q gen.Querier) Repository {
	return &pgRepository{q: q}
}

func (r *pgRepository) Create(ctx context.Context, l *Loan) (*Loan, error) {
	row, err := r.q.CreateLoan(ctx, gen.CreateLoanParams{
		UserID:             l.UserID,
		PeerID:             l.PeerID,
		PrincipalAmount:    toPgNumeric(l.PrincipalAmount),
		Currency:           l.Currency,
		InterestType:       string(l.InterestType),
		AnnualInterestRate: toPgNumeric(l.AnnualInterestRate),
		DayCountConvention: l.DayCountConvention,
		StartDate:          toPgDate(l.StartDate),
		DueDate:            toPgDatePtr(l.DueDate),
		Notes:              toPgText(l.Notes),
	})
	if err != nil {
		return nil, err
	}

	return toDomainLoan(row), nil
}

func (r *pgRepository) GetByIDAndUser(ctx context.Context, id, userID uuid.UUID) (*Loan, error) {
	row, err := r.q.GetLoanByIDAndUser(ctx, gen.GetLoanByIDAndUserParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, internalErrors.NewNotFoundError("loan not found")
		}
		return nil, err
	}

	return toDomainLoan(row), nil
}

func (r *pgRepository) ListByUser(ctx context.Context, userID uuid.UUID, peerID *uuid.UUID, status string, limit, offset int32) ([]*Loan, int64, error) {
	peerIDVal := uuid.Nil
	if peerID != nil {
		peerIDVal = *peerID
	}

	total, err := r.q.CountLoansByUser(ctx, gen.CountLoansByUserParams{
		UserID:  userID,
		Column2: peerIDVal,
		Column3: status,
	})
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.q.ListLoansByUser(ctx, gen.ListLoansByUserParams{
		UserID:  userID,
		Column2: peerIDVal,
		Column3: status,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, 0, err
	}

	loans := make([]*Loan, len(rows))
	for i, row := range rows {
		var peerEmail *string
		if row.PeerEmail.Valid {
			s := row.PeerEmail.String
			peerEmail = &s
		}

		l := &Loan{
			ID:                 row.ID,
			UserID:             row.UserID,
			PeerID:             row.PeerID,
			PeerName:           row.PeerName,
			PeerEmail:          peerEmail,
			PrincipalAmount:    fromPgNumeric(row.PrincipalAmount),
			Currency:           row.Currency,
			InterestType:       InterestType(row.InterestType),
			AnnualInterestRate: fromPgNumeric(row.AnnualInterestRate),
			DayCountConvention: row.DayCountConvention,
			StartDate:          row.StartDate.Time,
			DueDate:            fromPgDate(row.DueDate),
			Status:             Status(row.Status),
			Version:            int(row.Version),
			Notes:              fromPgText(row.Notes),
			CreatedAt:          row.CreatedAt.Time,
			UpdatedAt:          row.UpdatedAt.Time,
		}
		loans[i] = l
	}

	return loans, total, nil
}

func (r *pgRepository) UpdateDraft(ctx context.Context, l *Loan) (*Loan, error) {
	row, err := r.q.UpdateDraftLoan(ctx, gen.UpdateDraftLoanParams{
		ID:                 l.ID,
		UserID:             l.UserID,
		PeerID:             l.PeerID,
		PrincipalAmount:    toPgNumeric(l.PrincipalAmount),
		Currency:           l.Currency,
		InterestType:       string(l.InterestType),
		AnnualInterestRate: toPgNumeric(l.AnnualInterestRate),
		DayCountConvention: l.DayCountConvention,
		StartDate:          toPgDate(l.StartDate),
		DueDate:            toPgDatePtr(l.DueDate),
		Notes:              toPgText(l.Notes),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, internalErrors.NewNotFoundError("draft loan not found or not in draft status")
		}
		return nil, err
	}

	return toDomainLoan(row), nil
}

func (r *pgRepository) UpdateStatus(ctx context.Context, id, userID uuid.UUID, newStatus Status) (*Loan, error) {
	row, err := r.q.UpdateLoanStatus(ctx, gen.UpdateLoanStatusParams{
		ID:     id,
		UserID: userID,
		Status: string(newStatus),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, internalErrors.NewNotFoundError("loan not found")
		}
		return nil, err
	}

	return toDomainLoan(row), nil
}

func (r *pgRepository) Amend(ctx context.Context, id, userID uuid.UUID, principal, rate decimal.Decimal, interestType InterestType, dueDate *time.Time, notes *string) (*Loan, error) {
	row, err := r.q.AmendLoan(ctx, gen.AmendLoanParams{
		ID:                 id,
		UserID:             userID,
		PrincipalAmount:    toPgNumeric(principal),
		InterestType:       string(interestType),
		AnnualInterestRate: toPgNumeric(rate),
		DueDate:            toPgDatePtr(dueDate),
		Notes:              toPgText(notes),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, internalErrors.NewNotFoundError("active loan not found for amendment")
		}
		return nil, err
	}

	return toDomainLoan(row), nil
}

func (r *pgRepository) CreateLedgerEvent(ctx context.Context, event *LedgerEvent) (*LedgerEvent, error) {
	var loanIDPg pgtype.UUID
	if event.LoanID != nil {
		loanIDPg = pgtype.UUID{Bytes: *event.LoanID, Valid: true}
	}
	var actorIDPg pgtype.UUID
	if event.ActorID != nil {
		actorIDPg = pgtype.UUID{Bytes: *event.ActorID, Valid: true}
	}

	row, err := r.q.CreateLedgerEvent(ctx, gen.CreateLedgerEventParams{
		UserID:    event.UserID,
		LoanID:    loanIDPg,
		EventType: event.EventType,
		ActorType: event.ActorType,
		ActorID:   actorIDPg,
		Payload:   event.Payload,
	})
	if err != nil {
		return nil, err
	}

	return toDomainLedgerEvent(row), nil
}

func (r *pgRepository) ListLedgerEvents(ctx context.Context, loanID, userID uuid.UUID) ([]*LedgerEvent, error) {
	rows, err := r.q.ListLedgerEventsByLoan(ctx, gen.ListLedgerEventsByLoanParams{
		UserID: userID,
		LoanID: pgtype.UUID{Bytes: loanID, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	events := make([]*LedgerEvent, len(rows))
	for i, row := range rows {
		events[i] = toDomainLedgerEvent(row)
	}

	return events, nil
}

func toPgDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}

func toPgDatePtr(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{Valid: false}
	}
	return pgtype.Date{Time: *t, Valid: true}
}

func fromPgDate(d pgtype.Date) *time.Time {
	if !d.Valid {
		return nil
	}
	t := d.Time
	return &t
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

func toDomainLoan(row gen.Loans) *Loan {
	return &Loan{
		ID:                 row.ID,
		UserID:             row.UserID,
		PeerID:             row.PeerID,
		PrincipalAmount:    fromPgNumeric(row.PrincipalAmount),
		Currency:           row.Currency,
		InterestType:       InterestType(row.InterestType),
		AnnualInterestRate: fromPgNumeric(row.AnnualInterestRate),
		DayCountConvention: row.DayCountConvention,
		StartDate:          row.StartDate.Time,
		DueDate:            fromPgDate(row.DueDate),
		Status:             Status(row.Status),
		Version:            int(row.Version),
		Notes:              fromPgText(row.Notes),
		CreatedAt:          row.CreatedAt.Time,
		UpdatedAt:          row.UpdatedAt.Time,
	}
}

func toDomainLedgerEvent(row gen.LedgerEvents) *LedgerEvent {
	var loanID *uuid.UUID
	if row.LoanID.Valid {
		id := uuid.UUID(row.LoanID.Bytes)
		loanID = &id
	}
	var actorID *uuid.UUID
	if row.ActorID.Valid {
		id := uuid.UUID(row.ActorID.Bytes)
		actorID = &id
	}

	return &LedgerEvent{
		ID:        row.ID,
		UserID:    row.UserID,
		LoanID:    loanID,
		EventType: row.EventType,
		ActorType: row.ActorType,
		ActorID:   actorID,
		Payload:   json.RawMessage(row.Payload),
		CreatedAt: row.CreatedAt.Time,
	}
}
