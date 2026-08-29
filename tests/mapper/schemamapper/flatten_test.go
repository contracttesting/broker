package schemamapper_test

import (
	"testing"

	"github.com/contracttesting/broker/internal/dsl"
	"github.com/contracttesting/broker/internal/mapper/schemamapper"
	"github.com/contracttesting/broker/internal/model"
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

func TestToPropertyModels_OptionalRefSite_MakesTheNodeOptional(t *testing.T) {
	properties := schemamapper.ToPropertyModels(addressSchemas(false), userWithAddress(true))

	require.Contains(t, properties, "$.address")
	assert.Equal(t, model.Property{Path: "$.address", Type: "object", Optional: true}, properties["$.address"])
	assert.Equal(t, model.Property{Path: "$.address.street", Type: "string", Optional: false}, properties["$.address.street"])
}

func TestToPropertyModels_RequiredRefSite_ToRequiredTarget_StaysRequired(t *testing.T) {
	properties := schemamapper.ToPropertyModels(addressSchemas(false), userWithAddress(false))

	require.Contains(t, properties, "$.address")
	assert.Equal(t, model.Property{Path: "$.address", Type: "object", Optional: false}, properties["$.address"])
}

func TestToPropertyModels_RequiredRefSite_ToOptionalTarget_StaysOptional(t *testing.T) {
	properties := schemamapper.ToPropertyModels(addressSchemas(true), userWithAddress(false))

	require.Contains(t, properties, "$.address")
	assert.Equal(t, model.Property{Path: "$.address", Type: "object", Optional: true}, properties["$.address"])
}
