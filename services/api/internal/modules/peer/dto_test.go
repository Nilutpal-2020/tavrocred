package peer_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"tavrocred/services/api/internal/modules/peer"
)

func TestCreatePeerRequest_Validation(t *testing.T) {
	validName := "Alice Smith"
	validEmail := "alice@example.com"
	validPhone := "+919876543210"
	validNotes := "Friend from college"

	valid := peer.CreatePeerRequest{
		Name:  validName,
		Email: &validEmail,
		Phone: &validPhone,
		Notes: &validNotes,
	}
	assert.NoError(t, valid.SanitizeAndValidate())

	// Missing name
	missingName := peer.CreatePeerRequest{
		Name: "   ",
	}
	assert.Error(t, missingName.SanitizeAndValidate())

	// Name too long (> 255 chars)
	longName := peer.CreatePeerRequest{
		Name: strings.Repeat("A", 256),
	}
	assert.Error(t, longName.SanitizeAndValidate())

	// Invalid email
	invalidEmail := "not-an-email"
	badEmailReq := peer.CreatePeerRequest{
		Name:  validName,
		Email: &invalidEmail,
	}
	assert.Error(t, badEmailReq.SanitizeAndValidate())

	// Phone too long (> 50 chars)
	longPhone := strings.Repeat("1", 51)
	badPhoneReq := peer.CreatePeerRequest{
		Name:  validName,
		Phone: &longPhone,
	}
	assert.Error(t, badPhoneReq.SanitizeAndValidate())
}

func TestUpdatePeerRequest_Validation(t *testing.T) {
	valid := peer.UpdatePeerRequest{
		Name: "Bob Jones",
	}
	assert.NoError(t, valid.SanitizeAndValidate())

	blank := peer.UpdatePeerRequest{
		Name: "",
	}
	assert.Error(t, blank.SanitizeAndValidate())
}

func TestListPeersQuery_Pagination(t *testing.T) {
	// Defaults
	q := peer.ListPeersQuery{}
	q.Sanitize()
	assert.Equal(t, 1, q.Page)
	assert.Equal(t, 20, q.PageSize)
	assert.Equal(t, int32(0), q.Offset())
	assert.Equal(t, int32(20), q.Limit())

	// Custom valid page and page size
	q2 := peer.ListPeersQuery{
		Page:     3,
		PageSize: 15,
	}
	q2.Sanitize()
	assert.Equal(t, 3, q2.Page)
	assert.Equal(t, 15, q2.PageSize)
	assert.Equal(t, int32(30), q2.Offset()) // (3-1) * 15 = 30
	assert.Equal(t, int32(15), q2.Limit())

	// Page size capped at 100
	q3 := peer.ListPeersQuery{
		Page:     1,
		PageSize: 500,
	}
	q3.Sanitize()
	assert.Equal(t, 100, q3.PageSize)
}
