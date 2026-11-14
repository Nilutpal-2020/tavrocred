package borrower_test

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tavrocred/services/api/internal/modules/borrower"
)

func TestCreateBorrowerLinkRequest_Sanitize(t *testing.T) {
	t.Run("default hours", func(t *testing.T) {
		req := borrower.CreateBorrowerLinkRequest{ExpiresInHours: 0}
		assert.Equal(t, 72*time.Hour, req.Sanitize())
	})

	t.Run("custom valid hours", func(t *testing.T) {
		req := borrower.CreateBorrowerLinkRequest{ExpiresInHours: 48}
		assert.Equal(t, 48*time.Hour, req.Sanitize())
	})

	t.Run("max hours clamped", func(t *testing.T) {
		req := borrower.CreateBorrowerLinkRequest{ExpiresInHours: 1000}
		assert.Equal(t, 720*time.Hour, req.Sanitize())
	})
}

func TestBorrowerAccessRequest_Validate(t *testing.T) {
	t.Run("valid token", func(t *testing.T) {
		req := borrower.BorrowerAccessRequest{Token: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}
		assert.NoError(t, req.Validate())
	})

	t.Run("token too short", func(t *testing.T) {
		req := borrower.BorrowerAccessRequest{Token: "too-short"}
		assert.Error(t, req.Validate())
	})
}

func TestBorrowerDisputeRequest_Validate(t *testing.T) {
	t.Run("valid reason", func(t *testing.T) {
		req := borrower.BorrowerDisputeRequest{Reason: "Interest rate is different from verbal agreement"}
		assert.NoError(t, req.Validate())
	})

	t.Run("blank reason", func(t *testing.T) {
		req := borrower.BorrowerDisputeRequest{Reason: "   "}
		assert.Error(t, req.Validate())
	})
}

func TestBorrowerReportRepaymentRequest_SanitizeAndValidate(t *testing.T) {
	loanStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("valid input", func(t *testing.T) {
		req := borrower.BorrowerReportRepaymentRequest{
			Amount:            "2500.00",
			PaymentDate:       "2026-01-10",
			PaymentMethod:     "upi",
			ExternalReference: strPtr("UPI-123456"),
			Notes:             strPtr("Sent via PhonePe"),
		}

		amt, payDate, err := req.SanitizeAndValidate(loanStart)
		require.NoError(t, err)
		assert.True(t, amt.Equal(decimal.NewFromInt(2500)))
		assert.Equal(t, "UPI", req.PaymentMethod)
		assert.Equal(t, "UPI-123456", *req.ExternalReference)
		assert.Equal(t, "Sent via PhonePe", *req.Notes)
		assert.Equal(t, 2026, payDate.Year())
	})

	t.Run("negative amount", func(t *testing.T) {
		req := borrower.BorrowerReportRepaymentRequest{
			Amount:        "-500",
			PaymentDate:   "2026-01-10",
			PaymentMethod: "UPI",
		}
		_, _, err := req.SanitizeAndValidate(loanStart)
		assert.Error(t, err)
	})

	t.Run("payment date before loan start", func(t *testing.T) {
		req := borrower.BorrowerReportRepaymentRequest{
			Amount:        "500",
			PaymentDate:   "2025-12-15",
			PaymentMethod: "CASH",
		}
		_, _, err := req.SanitizeAndValidate(loanStart)
		assert.Error(t, err)
	})

	t.Run("invalid payment method", func(t *testing.T) {
		req := borrower.BorrowerReportRepaymentRequest{
			Amount:        "500",
			PaymentDate:   "2026-01-10",
			PaymentMethod: "BITCOIN",
		}
		_, _, err := req.SanitizeAndValidate(loanStart)
		assert.Error(t, err)
	})
}

func strPtr(s string) *string {
	return &s
}
