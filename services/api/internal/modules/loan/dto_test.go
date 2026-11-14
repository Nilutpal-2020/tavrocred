package loan_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"tavrocred/services/api/internal/modules/loan"
)

func TestCreateLoanRequest_Validation(t *testing.T) {
	validPeer := uuid.New()
	validStart := "2026-01-01"
	validDue := "2026-12-31"

	// Valid NONE interest
	reqNone := loan.CreateLoanRequest{
		PeerID:             validPeer,
		PrincipalAmount:    "10000.00",
		Currency:           "inr",
		InterestType:       "none",
		AnnualInterestRate: "0",
		StartDate:          validStart,
		DueDate:            &validDue,
	}
	p, r, start, due, err := reqNone.SanitizeAndValidate()
	assert.NoError(t, err)
	assert.Equal(t, "10000.00", p.StringFixed(2))
	assert.Equal(t, "0.00", r.StringFixed(2))
	assert.Equal(t, "INR", reqNone.Currency)
	assert.Equal(t, "NONE", reqNone.InterestType)
	assert.NotNil(t, due)
	assert.Equal(t, 2026, start.Year())

	// Valid SIMPLE interest
	reqSimple := loan.CreateLoanRequest{
		PeerID:             validPeer,
		PrincipalAmount:    "50000.50",
		InterestType:       "SIMPLE",
		AnnualInterestRate: "12.50",
		StartDate:          validStart,
	}
	p, r, _, _, err = reqSimple.SanitizeAndValidate()
	assert.NoError(t, err)
	assert.Equal(t, "50000.50", p.StringFixed(2))
	assert.Equal(t, "12.50", r.StringFixed(2))
	assert.Equal(t, "INR", reqSimple.Currency)

	// Invalid: Non-zero rate with NONE interest
	reqBadRateNone := loan.CreateLoanRequest{
		PeerID:             validPeer,
		PrincipalAmount:    "10000.00",
		InterestType:       "NONE",
		AnnualInterestRate: "5.00",
		StartDate:          validStart,
	}
	_, _, _, _, err = reqBadRateNone.SanitizeAndValidate()
	assert.Error(t, err)

	// Invalid: Negative principal
	reqNegPrincipal := loan.CreateLoanRequest{
		PeerID:          validPeer,
		PrincipalAmount: "-500.00",
		InterestType:    "NONE",
		StartDate:       validStart,
	}
	_, _, _, _, err = reqNegPrincipal.SanitizeAndValidate()
	assert.Error(t, err)

	// Invalid: Due date earlier than start date
	earlierDue := "2025-12-31"
	reqEarlyDue := loan.CreateLoanRequest{
		PeerID:          validPeer,
		PrincipalAmount: "10000.00",
		InterestType:    "NONE",
		StartDate:       validStart,
		DueDate:         &earlierDue,
	}
	_, _, _, _, err = reqEarlyDue.SanitizeAndValidate()
	assert.Error(t, err)

	// Invalid: Missing peer ID
	reqNoPeer := loan.CreateLoanRequest{
		PeerID:          uuid.Nil,
		PrincipalAmount: "10000.00",
		InterestType:    "NONE",
		StartDate:       validStart,
	}
	_, _, _, _, err = reqNoPeer.SanitizeAndValidate()
	assert.Error(t, err)
}

func TestAmendLoanRequest_Validation(t *testing.T) {
	startDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	validDue := "2026-12-31"

	valid := loan.AmendLoanRequest{
		PrincipalAmount:    "12000.00",
		InterestType:       "SIMPLE",
		AnnualInterestRate: "10.00",
		DueDate:            &validDue,
		Reason:             "Mutually agreed interest reduction",
	}
	p, r, due, err := valid.SanitizeAndValidate(startDate)
	assert.NoError(t, err)
	assert.Equal(t, "12000.00", p.StringFixed(2))
	assert.Equal(t, "10.00", r.StringFixed(2))
	assert.NotNil(t, due)

	// Missing reason
	missingReason := loan.AmendLoanRequest{
		PrincipalAmount: "12000.00",
		InterestType:    "NONE",
		Reason:          "   ",
	}
	_, _, _, err = missingReason.SanitizeAndValidate(startDate)
	assert.Error(t, err)
}
