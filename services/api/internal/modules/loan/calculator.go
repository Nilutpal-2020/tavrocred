package loan

import (
	"time"

	"github.com/shopspring/decimal"
)

type FinancialSummary struct {
	PrincipalAmount          decimal.Decimal `json:"principal_amount"`
	InterestAccrued          decimal.Decimal `json:"interest_accrued"`
	TotalAmountDue           decimal.Decimal `json:"total_amount_due"`
	TotalConfirmedRepayments decimal.Decimal `json:"total_confirmed_repayments"`
	OutstandingBalance       decimal.Decimal `json:"outstanding_balance"`
	ElapsedDays              int64           `json:"elapsed_days"`
	CalculationDate          time.Time       `json:"calculation_date"`
}

type Calculator struct{}

func NewCalculator() *Calculator {
	return &Calculator{}
}

func (c *Calculator) CalculateSummary(
	principal decimal.Decimal,
	interestType string,
	annualRate decimal.Decimal,
	startDate time.Time,
	asOfDate time.Time,
	confirmedRepayments decimal.Decimal,
) FinancialSummary {
	// Truncate times to calendar dates (UTC midnight) for day count
	startCal := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
	asOfCal := time.Date(asOfDate.Year(), asOfDate.Month(), asOfDate.Day(), 0, 0, 0, 0, time.UTC)

	var elapsedDays int64
	if asOfCal.After(startCal) {
		duration := asOfCal.Sub(startCal)
		elapsedDays = int64(duration.Hours() / 24)
	}

	interest := decimal.Zero
	if interestType == "SIMPLE" && annualRate.GreaterThan(decimal.Zero) && elapsedDays > 0 {
		// interest = principal * (annualRate / 100) * (elapsedDays / 365)
		rateFraction := annualRate.Div(decimal.NewFromInt(100))
		daysFraction := decimal.NewFromInt(elapsedDays).Div(decimal.NewFromInt(365))
		interest = principal.Mul(rateFraction).Mul(daysFraction).Round(2)
	} else {
		interest = decimal.Zero.Round(2)
	}

	principal2Dec := principal.Round(2)
	totalDue := principal2Dec.Add(interest).Round(2)
	repayments2Dec := confirmedRepayments.Round(2)
	outstanding := totalDue.Sub(repayments2Dec).Round(2)

	return FinancialSummary{
		PrincipalAmount:          principal2Dec,
		InterestAccrued:          interest,
		TotalAmountDue:           totalDue,
		TotalConfirmedRepayments: repayments2Dec,
		OutstandingBalance:       outstanding,
		ElapsedDays:              elapsedDays,
		CalculationDate:          asOfCal,
	}
}
