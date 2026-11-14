package loan

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	internalErrors "tavrocred/services/api/internal/errors"
)

type CreateLoanRequest struct {
	PeerID             uuid.UUID `json:"peer_id"`
	PrincipalAmount    string    `json:"principal_amount"`
	Currency           string    `json:"currency"`
	InterestType       string    `json:"interest_type"`
	AnnualInterestRate string    `json:"annual_interest_rate"`
	StartDate          string    `json:"start_date"`
	DueDate            *string   `json:"due_date,omitempty"`
	Notes              *string   `json:"notes,omitempty"`
}

func (r *CreateLoanRequest) SanitizeAndValidate() (decimal.Decimal, decimal.Decimal, time.Time, *time.Time, error) {
	details := make(map[string]string)

	if r.PeerID == uuid.Nil {
		details["peer_id"] = "valid peer_id is required"
	}

	principal, err := decimal.NewFromString(strings.TrimSpace(r.PrincipalAmount))
	if err != nil || principal.LessThanOrEqual(decimal.Zero) {
		details["principal_amount"] = "principal_amount must be a valid positive number"
	}

	r.Currency = strings.ToUpper(strings.TrimSpace(r.Currency))
	if r.Currency == "" {
		r.Currency = "INR"
	} else if len(r.Currency) != 3 {
		details["currency"] = "currency must be a 3-letter ISO code (e.g. INR)"
	}

	r.InterestType = strings.ToUpper(strings.TrimSpace(r.InterestType))
	if r.InterestType != "NONE" && r.InterestType != "SIMPLE" {
		details["interest_type"] = "interest_type must be either NONE or SIMPLE"
	}

	rate := decimal.Zero
	if r.AnnualInterestRate != "" {
		parsedRate, err := decimal.NewFromString(strings.TrimSpace(r.AnnualInterestRate))
		if err != nil || parsedRate.LessThan(decimal.Zero) {
			details["annual_interest_rate"] = "annual_interest_rate must be a valid non-negative number"
		} else {
			rate = parsedRate
		}
	}

	if r.InterestType == "NONE" && !rate.IsZero() {
		details["annual_interest_rate"] = "annual_interest_rate must be 0 for interest_type NONE"
	}

	startDate, err := time.Parse("2006-01-02", strings.TrimSpace(r.StartDate))
	if err != nil {
		details["start_date"] = "start_date must be in YYYY-MM-DD format"
	}

	var dueDate *time.Time
	if r.DueDate != nil && strings.TrimSpace(*r.DueDate) != "" {
		parsedDue, err := time.Parse("2006-01-02", strings.TrimSpace(*r.DueDate))
		if err != nil {
			details["due_date"] = "due_date must be in YYYY-MM-DD format"
		} else if !parsedDue.IsZero() && parsedDue.Before(startDate) {
			details["due_date"] = "due_date cannot be earlier than start_date"
		} else {
			dueDate = &parsedDue
		}
	}

	if r.Notes != nil {
		clean := strings.TrimSpace(*r.Notes)
		if clean == "" {
			r.Notes = nil
		} else {
			r.Notes = &clean
		}
	}

	if len(details) > 0 {
		return decimal.Zero, decimal.Zero, time.Time{}, nil, internalErrors.NewValidationError("invalid loan creation input", details)
	}

	return principal.Round(2), rate.Round(2), startDate, dueDate, nil
}

type UpdateLoanRequest struct {
	PeerID             uuid.UUID `json:"peer_id"`
	PrincipalAmount    string    `json:"principal_amount"`
	Currency           string    `json:"currency"`
	InterestType       string    `json:"interest_type"`
	AnnualInterestRate string    `json:"annual_interest_rate"`
	StartDate          string    `json:"start_date"`
	DueDate            *string   `json:"due_date,omitempty"`
	Notes              *string   `json:"notes,omitempty"`
}

func (r *UpdateLoanRequest) SanitizeAndValidate() (decimal.Decimal, decimal.Decimal, time.Time, *time.Time, error) {
	req := CreateLoanRequest{
		PeerID:             r.PeerID,
		PrincipalAmount:    r.PrincipalAmount,
		Currency:           r.Currency,
		InterestType:       r.InterestType,
		AnnualInterestRate: r.AnnualInterestRate,
		StartDate:          r.StartDate,
		DueDate:            r.DueDate,
		Notes:              r.Notes,
	}
	return req.SanitizeAndValidate()
}

type AmendLoanRequest struct {
	PrincipalAmount    string  `json:"principal_amount"`
	InterestType       string  `json:"interest_type"`
	AnnualInterestRate string  `json:"annual_interest_rate"`
	DueDate            *string `json:"due_date,omitempty"`
	Notes              *string `json:"notes,omitempty"`
	Reason             string  `json:"reason"`
}

func (r *AmendLoanRequest) SanitizeAndValidate(startDate time.Time) (decimal.Decimal, decimal.Decimal, *time.Time, error) {
	details := make(map[string]string)

	principal, err := decimal.NewFromString(strings.TrimSpace(r.PrincipalAmount))
	if err != nil || principal.LessThanOrEqual(decimal.Zero) {
		details["principal_amount"] = "principal_amount must be a valid positive number"
	}

	r.InterestType = strings.ToUpper(strings.TrimSpace(r.InterestType))
	if r.InterestType != "NONE" && r.InterestType != "SIMPLE" {
		details["interest_type"] = "interest_type must be either NONE or SIMPLE"
	}

	rate := decimal.Zero
	if r.AnnualInterestRate != "" {
		parsedRate, err := decimal.NewFromString(strings.TrimSpace(r.AnnualInterestRate))
		if err != nil || parsedRate.LessThan(decimal.Zero) {
			details["annual_interest_rate"] = "annual_interest_rate must be a valid non-negative number"
		} else {
			rate = parsedRate
		}
	}

	if r.InterestType == "NONE" && !rate.IsZero() {
		details["annual_interest_rate"] = "annual_interest_rate must be 0 for interest_type NONE"
	}

	var dueDate *time.Time
	if r.DueDate != nil && strings.TrimSpace(*r.DueDate) != "" {
		parsedDue, err := time.Parse("2006-01-02", strings.TrimSpace(*r.DueDate))
		if err != nil {
			details["due_date"] = "due_date must be in YYYY-MM-DD format"
		} else if !parsedDue.IsZero() && parsedDue.Before(startDate) {
			details["due_date"] = "due_date cannot be earlier than start_date"
		} else {
			dueDate = &parsedDue
		}
	}

	r.Reason = strings.TrimSpace(r.Reason)
	if r.Reason == "" {
		details["reason"] = "amendment reason is required"
	}

	if r.Notes != nil {
		clean := strings.TrimSpace(*r.Notes)
		if clean == "" {
			r.Notes = nil
		} else {
			r.Notes = &clean
		}
	}

	if len(details) > 0 {
		return decimal.Zero, decimal.Zero, nil, internalErrors.NewValidationError("invalid loan amendment input", details)
	}

	return principal.Round(2), rate.Round(2), dueDate, nil
}

type CancelLoanRequest struct {
	Reason string `json:"reason"`
}

type DisputeLoanRequest struct {
	Reason string `json:"reason"`
}

type FinancialSummaryResponse struct {
	PrincipalAmount          string `json:"principal_amount"`
	InterestAccrued          string `json:"interest_accrued"`
	TotalAmountDue           string `json:"total_amount_due"`
	TotalConfirmedRepayments string `json:"total_confirmed_repayments"`
	OutstandingBalance       string `json:"outstanding_balance"`
	ElapsedDays              int64  `json:"elapsed_days"`
	CalculationDate          string `json:"calculation_date"`
}

type LoanResponse struct {
	ID                 uuid.UUID                `json:"id"`
	UserID             uuid.UUID                `json:"user_id"`
	PeerID             uuid.UUID                `json:"peer_id"`
	PeerName           string                   `json:"peer_name,omitempty"`
	PeerEmail          *string                  `json:"peer_email,omitempty"`
	PrincipalAmount    string                   `json:"principal_amount"`
	Currency           string                   `json:"currency"`
	InterestType       string                   `json:"interest_type"`
	AnnualInterestRate string                   `json:"annual_interest_rate"`
	DayCountConvention string                   `json:"day_count_convention"`
	StartDate          string                   `json:"start_date"`
	DueDate            *string                  `json:"due_date,omitempty"`
	Status             string                   `json:"status"`
	Version            int                      `json:"version"`
	Notes              *string                  `json:"notes,omitempty"`
	FinancialSummary   FinancialSummaryResponse `json:"financial_summary"`
	CreatedAt          time.Time                `json:"created_at"`
	UpdatedAt          time.Time                `json:"updated_at"`
}

func ToLoanResponse(l *Loan, summary FinancialSummary) LoanResponse {
	var dueDateStr *string
	if l.DueDate != nil {
		s := l.DueDate.Format("2006-01-02")
		dueDateStr = &s
	}

	return LoanResponse{
		ID:                 l.ID,
		UserID:             l.UserID,
		PeerID:             l.PeerID,
		PeerName:           l.PeerName,
		PeerEmail:          l.PeerEmail,
		PrincipalAmount:    l.PrincipalAmount.StringFixed(2),
		Currency:           l.Currency,
		InterestType:       string(l.InterestType),
		AnnualInterestRate: l.AnnualInterestRate.StringFixed(2),
		DayCountConvention: l.DayCountConvention,
		StartDate:          l.StartDate.Format("2006-01-02"),
		DueDate:            dueDateStr,
		Status:             string(l.Status),
		Version:            l.Version,
		Notes:              l.Notes,
		FinancialSummary: FinancialSummaryResponse{
			PrincipalAmount:          summary.PrincipalAmount.StringFixed(2),
			InterestAccrued:          summary.InterestAccrued.StringFixed(2),
			TotalAmountDue:           summary.TotalAmountDue.StringFixed(2),
			TotalConfirmedRepayments: summary.TotalConfirmedRepayments.StringFixed(2),
			OutstandingBalance:       summary.OutstandingBalance.StringFixed(2),
			ElapsedDays:              summary.ElapsedDays,
			CalculationDate:          summary.CalculationDate.Format("2006-01-02"),
		},
		CreatedAt: l.CreatedAt,
		UpdatedAt: l.UpdatedAt,
	}
}

type ListLoansQuery struct {
	Page     int
	PageSize int
	PeerID   *uuid.UUID
	Status   string
}

func (q *ListLoansQuery) Sanitize() {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = 20
	} else if q.PageSize > 100 {
		q.PageSize = 100
	}
	q.Status = strings.ToUpper(strings.TrimSpace(q.Status))
}

func (q *ListLoansQuery) Offset() int32 {
	return int32((q.Page - 1) * q.PageSize)
}

func (q *ListLoansQuery) Limit() int32 {
	return int32(q.PageSize)
}
