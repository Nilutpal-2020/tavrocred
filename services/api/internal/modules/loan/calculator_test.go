package loan_test

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"tavrocred/services/api/internal/modules/loan"
)

func TestCalculator_InterestFreeLoan(t *testing.T) {
	calc := loan.NewCalculator()

	startDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	asOfDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	principal, _ := decimal.NewFromString("50000.00")
	rate := decimal.Zero
	repayments := decimal.Zero

	summary := calc.CalculateSummary(principal, "NONE", rate, startDate, asOfDate, repayments)

	assert.Equal(t, "50000.00", summary.PrincipalAmount.StringFixed(2))
	assert.Equal(t, "0.00", summary.InterestAccrued.StringFixed(2))
	assert.Equal(t, "50000.00", summary.TotalAmountDue.StringFixed(2))
	assert.Equal(t, "50000.00", summary.OutstandingBalance.StringFixed(2))
	assert.Equal(t, int64(181), summary.ElapsedDays)
}

func TestCalculator_SimpleInterest_ExactOneYear(t *testing.T) {
	calc := loan.NewCalculator()

	// 365 days elapsed in non-leap year 2025
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	asOfDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) // 365 days
	principal, _ := decimal.NewFromString("10000.00")
	rate, _ := decimal.NewFromString("12.00") // 12% p.a.
	repayments := decimal.Zero

	summary := calc.CalculateSummary(principal, "SIMPLE", rate, startDate, asOfDate, repayments)

	assert.Equal(t, int64(365), summary.ElapsedDays)
	// 10000 * 0.12 * (365 / 365) = 1200.00
	assert.Equal(t, "1200.00", summary.InterestAccrued.StringFixed(2))
	assert.Equal(t, "11200.00", summary.TotalAmountDue.StringFixed(2))
	assert.Equal(t, "11200.00", summary.OutstandingBalance.StringFixed(2))
}

func TestCalculator_SimpleInterest_WithRepayments(t *testing.T) {
	calc := loan.NewCalculator()

	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	asOfDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	principal, _ := decimal.NewFromString("10000.00")
	rate, _ := decimal.NewFromString("12.00")
	repayments, _ := decimal.NewFromString("4000.00")

	summary := calc.CalculateSummary(principal, "SIMPLE", rate, startDate, asOfDate, repayments)

	assert.Equal(t, "1200.00", summary.InterestAccrued.StringFixed(2))
	assert.Equal(t, "11200.00", summary.TotalAmountDue.StringFixed(2))
	assert.Equal(t, "4000.00", summary.TotalConfirmedRepayments.StringFixed(2))
	// 11200.00 - 4000.00 = 7200.00
	assert.Equal(t, "7200.00", summary.OutstandingBalance.StringFixed(2))
}

func TestCalculator_LeapYearAndDateBoundaries(t *testing.T) {
	calc := loan.NewCalculator()

	// 2024 was a leap year (February had 29 days)
	startDate := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	asOfDate := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	principal, _ := decimal.NewFromString("36500.00")
	rate, _ := decimal.NewFromString("10.00") // 10%

	summary := calc.CalculateSummary(principal, "SIMPLE", rate, startDate, asOfDate, decimal.Zero)

	// In 2024, Feb 1 to Mar 1 is 29 days
	assert.Equal(t, int64(29), summary.ElapsedDays)
	// 36500 * 0.10 * (29 / 365) = 290.00
	assert.Equal(t, "290.00", summary.InterestAccrued.StringFixed(2))
	assert.Equal(t, "36790.00", summary.TotalAmountDue.StringFixed(2))
}

func TestCalculator_AsOfDateBeforeStartDate(t *testing.T) {
	calc := loan.NewCalculator()

	startDate := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	asOfDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	principal, _ := decimal.NewFromString("25000.00")
	rate, _ := decimal.NewFromString("15.00")

	summary := calc.CalculateSummary(principal, "SIMPLE", rate, startDate, asOfDate, decimal.Zero)

	assert.Equal(t, int64(0), summary.ElapsedDays)
	assert.Equal(t, "0.00", summary.InterestAccrued.StringFixed(2))
	assert.Equal(t, "25000.00", summary.TotalAmountDue.StringFixed(2))
	assert.Equal(t, "25000.00", summary.OutstandingBalance.StringFixed(2))
}
