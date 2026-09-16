package violation_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/contracttesting/broker/internal/features/publish_contract/violation"
)

func TestViolation_SerializaEmCamelCase(t *testing.T) {
	subject := violation.Violation{
		Code:    "key.unknown",
		Path:    "provides;rest;/pets;patch",
		Source:  "api.yaml",
		Details: map[string]string{"key": "patch"},
	}

	raw, err := json.Marshal(subject)

	require.NoError(t, err)
	assert.JSONEq(t, `{
		"code": "key.unknown",
		"path": "provides;rest;/pets;patch",
		"source": "api.yaml",
		"details": {"key": "patch"}
	}`, string(raw))
}
