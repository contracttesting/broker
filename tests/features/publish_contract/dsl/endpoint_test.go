package dsl_test

import (
	"testing"

	"github.com/contracttesting/broker/internal/features/publish_contract/dsl"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeEndpoint_TrimsTrailingSlash(t *testing.T) {
	assert.Equal(t, "/users", dsl.NormalizeEndpoint("/users/"))
}

func TestNormalizeEndpoint_KeepsRoot(t *testing.T) {
	assert.Equal(t, "/", dsl.NormalizeEndpoint("/"))
}

func TestNormalizeEndpoint_LeavesPlainEndpointAlone(t *testing.T) {
	assert.Equal(t, "/users", dsl.NormalizeEndpoint("/users"))
}
