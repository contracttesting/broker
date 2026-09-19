package contract_test

import (
	"testing"

	"github.com/contracttesting/broker/internal/features/publish_contract/contract"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeEndpoint_TrimsTrailingSlash(t *testing.T) {
	assert.Equal(t, "/users", contract.NormalizeEndpoint("/users/"))
}

func TestNormalizeEndpoint_KeepsRoot(t *testing.T) {
	assert.Equal(t, "/", contract.NormalizeEndpoint("/"))
}

func TestNormalizeEndpoint_LeavesPlainEndpointAlone(t *testing.T) {
	assert.Equal(t, "/users", contract.NormalizeEndpoint("/users"))
}
