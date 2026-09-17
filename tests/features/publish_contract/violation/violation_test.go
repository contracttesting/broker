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

func TestSortedByLocation_OrdenaPelaOrigemDepoisPeloCaminho(t *testing.T) {
	violations := []violation.Violation{
		{Code: "schema.duplicate", Path: "a", Source: "b.yaml"},
		{Code: "schema.duplicate", Path: "z", Source: "a.yaml"},
	}

	assert.Equal(t, []violation.Violation{
		{Code: "schema.duplicate", Path: "z", Source: "a.yaml"},
		{Code: "schema.duplicate", Path: "a", Source: "b.yaml"},
	}, violation.SortedByLocation(violations))
}

func TestSortedByLocation_EmpateMantemAOrdemDeEntrada(t *testing.T) {
	violations := []violation.Violation{
		{Code: "schema.duplicate", Path: "schemas;Pet", Source: "a.yaml"},
		{Code: "resource.duplicate", Path: "provides;rest;/pets;get;responses;200", Source: "a.yaml"},
		{Code: "schema.unresolved_name", Path: "provides;rest;/pets;get;responses;200", Source: "a.yaml"},
	}

	assert.Equal(t, []violation.Violation{
		{Code: "resource.duplicate", Path: "provides;rest;/pets;get;responses;200", Source: "a.yaml"},
		{Code: "schema.unresolved_name", Path: "provides;rest;/pets;get;responses;200", Source: "a.yaml"},
		{Code: "schema.duplicate", Path: "schemas;Pet", Source: "a.yaml"},
	}, violation.SortedByLocation(violations))
}

func TestSortedByLocation_NaoAlteraAEntrada(t *testing.T) {
	violations := []violation.Violation{{Source: "b.yaml"}, {Source: "a.yaml"}}

	violation.SortedByLocation(violations)

	assert.Equal(t, []violation.Violation{{Source: "b.yaml"}, {Source: "a.yaml"}}, violations)
}
