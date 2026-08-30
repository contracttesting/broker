package validator_test

import (
	"encoding/json"
	"testing"

	"github.com/contracttesting/broker/internal/features/publish_contract/dsl"
	"github.com/contracttesting/broker/internal/features/publish_contract/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func primitiveTypeSchemasJSON(schemaType string) string {
	return `{
  "schemas": {
    "Pet": {
      "type": "object",
      "properties": { "weight": { "type": "` + schemaType + `" } }
    }
  }
}`
}

func primitiveTypeViolations(t *testing.T, raw string) []string {
	t.Helper()

	var contract dsl.Contract
	require.NoError(t, json.Unmarshal([]byte(raw), &contract))

	fragments := []dsl.Fragment{{Source: "schemas.json", Contract: &contract}}

	return validator.NewContextualValidator().Validate(fragments)
}

func TestPrimitiveType_Accepts(t *testing.T) {
	for _, schemaType := range []string{"string", "integer", "float", "boolean"} {
		t.Run(schemaType, func(t *testing.T) {
			assert.Empty(t, primitiveTypeViolations(t, primitiveTypeSchemasJSON(schemaType)))
		})
	}
}

// numeric subtyping is deferred to a future `numeric` type; until then number is
// no more a type than any other misspelling
func TestPrimitiveType_RejectsNumber(t *testing.T) {
	violations := primitiveTypeViolations(t, primitiveTypeSchemasJSON("number"))

	require.Len(t, violations, 1)
	assert.Equal(t, `invalid schema type "number" at Pet.weight (schemas.json)`, violations[0])
}
