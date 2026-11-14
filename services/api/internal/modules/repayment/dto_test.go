package repayment_test

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internalErrors "tavrocred/services/api/internal/errors"
	"tavrocred/services/api/internal/modules/repayment"
)

func TestRecordRepaymentRequest_SanitizeAndValidate(t *testing.T) {
	loanStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("valid input", func(t *testing.T) {
		req := repayment.RecordRepaymentRequest{
			Amount:            " 5000.50 ",
			Currency:          " inr ",
			PaymentDate:       "2026-01-15",
			PaymentMethod:     " upi ",
			ExternalReference: strPtr(" TXN-12345 "),
			Notes:             strPtr(" First installment "),
		}

		amount, payDate, err := req.SanitizeAndValidate(loanStart)
		require.NoError(t, err)
		assert.True(t, amount.Equal(decimal.NewFromFloat(5000.50)))
		assert.Equal(t, "INR", req.Currency)
		assert.Equal(t, "UPI", req.PaymentMethod)
		assert.Equal(t, "TXN-12345", *req.ExternalReference)
		assert.Equal(t, "First installment", *req.Notes)
		assert.Equal(t, 2026, payDate.Year())
		assert.Equal(t, time.Month(1), payDate.Month())
		assert.Equal(t, 15, payDate.Day())
	})

	t.Run("default currency when empty", func(t *testing.T) {
		req := repayment.RecordRepaymentRequest{
			Amount:        "1000",
			PaymentDate:   "2026-01-10",
			PaymentMethod: "CASH",
		}

		amount, _, err := req.SanitizeAndValidate(loanStart)
		require.NoError(t, err)
		assert.True(t, amount.Equal(decimal.NewFromInt(1000)))
		assert.Equal(t, "INR", req.Currency)
	})

	t.Run("invalid amount", func(t *testing.T) {
		testCases := []string{"", "0", "-100", "invalid"}
		for _, tc := range testCases {
			req := repayment.RecordRepaymentRequest{
				Amount:        tc,
				PaymentDate:   "2026-01-10",
				PaymentMethod: "BANK_TRANSFER",
			}
			_, _, err := req.SanitizeAndValidate(loanStart)
			require.Error(t, err)
			var valErr *internalErrors.AppError
			require.ErrorAs(t, err, &valErr)
			assert.Contains(t, valErr.Details, "amount")
		}
	})

	t.Run("payment date before loan start date", func(t *testing.T) {
		req := repayment.RecordRepaymentRequest{
			Amount:        "500",
			PaymentDate:   "2025-12-31",
			PaymentMethod: "CASH",
		}
		_, _, err := req.SanitizeAndValidate(loanStart)
		require.Error(t, err)
		var valErr *internalErrors.AppError
		require.ErrorAs(t, err, &valErr)
		assert.Contains(t, valErr.Details, "payment_date")
	})

	t.Run("payment date far in the future", func(t *testing.T) {
		futureDate := time.Now().UTC().Add(48 * time.Hour).Format("2006-01-02")
		req := repayment.RecordRepaymentRequest{
			Amount:        "500",
			PaymentDate:   futureDate,
			PaymentMethod: "CASH",
		}
		_, _, err := req.SanitizeAndValidate(loanStart)
		require.Error(t, err)
		var valErr *internalErrors.AppError
		require.ErrorAs(t, err, &valErr)
		assert.Contains(t, valErr.Details, "payment_date")
	})

	t.Run("invalid payment method", func(t *testing.T) {
		req := repayment.RecordRepaymentRequest{
			Amount:        "500",
			PaymentDate:   "2026-01-10",
			PaymentMethod: "CRYPTO",
		}
		_, _, err := req.SanitizeAndValidate(loanStart)
		require.Error(t, err)
		var valErr *internalErrors.AppError
		require.ErrorAs(t, err, &valErr)
		assert.Contains(t, valErr.Details, "payment_method")
	})
}

func TestRejectRepaymentRequest_Validate(t *testing.T) {
	t.Run("valid reason", func(t *testing.T) {
		req := repayment.RejectRepaymentRequest{Reason: "Did not receive funds in bank"}
		err := req.Validate()
		require.NoError(t, err)
	})

	t.Run("blank reason", func(t *testing.T) {
		req := repayment.RejectRepaymentRequest{Reason: "   "}
		err := req.Validate()
		require.Error(t, err)
	})
}

func TestReverseRepaymentRequest_Validate(t *testing.T) {
	t.Run("valid reason", func(t *testing.T) {
		req := repayment.ReverseRepaymentRequest{Reason: "Bank chargeback"}
		err := req.Validate()
		require.NoError(t, err)
	})

	t.Run("blank reason", func(t *testing.T) {
		req := repayment.ReverseRepaymentRequest{Reason: " "}
		err := req.Validate()
		require.Error(t, err)
	})
}

func TestListRepaymentsQuery_Sanitize(t *testing.T) {
	q := repayment.ListRepaymentsQuery{
		Page:     -1,
		PageSize: 500,
		Status:   " confirmed ",
	}
	q.Sanitize()
	assert.Equal(t, 1, q.Page)
	assert.Equal(t, 100, q.PageSize)
	assert.Equal(t, "CONFIRMED", q.Status)
	assert.Equal(t, int32(0), q.Offset())
	assert.Equal(t, int32(100), q.Limit())
}

func strPtr(s string) *string {
	return &s
}
