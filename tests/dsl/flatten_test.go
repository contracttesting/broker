package dsl_test

import (
	"testing"

	"github.com/contracttesting/broker/internal/dsl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addressSchemas(targetOptional bool) dsl.SchemasMap {
	return dsl.SchemasMap{
		"Address": {
			Type:     "object",
			Optional: targetOptional,
			Properties: map[string]dsl.Schema{
				"street": {Type: "string"},
			},
		},
	}
}

func userWithAddress(siteOptional bool) dsl.Schema {
	return dsl.Schema{
		Type: "object",
		Properties: map[string]dsl.Schema{
			"address": {Ref: "Address", Optional: siteOptional},
		},
	}
}

func TestFlattenSchema_OptionalRefSite_MakesTheNodeOptional(t *testing.T) {
	flattened := dsl.FlattenSchema(addressSchemas(false), userWithAddress(true))

	require.Contains(t, flattened, "$.address")
	assert.Equal(t, dsl.FlatProperty{Type: "object", Optional: true}, flattened["$.address"])
	assert.Equal(t, dsl.FlatProperty{Type: "string", Optional: false}, flattened["$.address.street"])
}

func TestFlattenSchema_RequiredRefSite_ToRequiredTarget_StaysRequired(t *testing.T) {
	flattened := dsl.FlattenSchema(addressSchemas(false), userWithAddress(false))

	require.Contains(t, flattened, "$.address")
	assert.Equal(t, dsl.FlatProperty{Type: "object", Optional: false}, flattened["$.address"])
}

func TestFlattenSchema_RequiredRefSite_ToOptionalTarget_StaysOptional(t *testing.T) {
	flattened := dsl.FlattenSchema(addressSchemas(true), userWithAddress(false))

	require.Contains(t, flattened, "$.address")
	assert.Equal(t, dsl.FlatProperty{Type: "object", Optional: true}, flattened["$.address"])
}
