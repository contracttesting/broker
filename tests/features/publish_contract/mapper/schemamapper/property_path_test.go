package schemamapper_test

import (
	"testing"

	"github.com/contracttesting/broker/internal/features/publish_contract/dsl"
	"github.com/contracttesting/broker/internal/features/publish_contract/mapper/schemamapper"
	"github.com/contracttesting/broker/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestToPropertyModels_Root_IsDollar(t *testing.T) {
	properties := schemamapper.ToPropertyModels(nil, dsl.Schema{Type: "string"})

	assert.Equal(t, map[string]model.Property{
		"$": {Path: "$", Type: "string", Optional: false},
	}, properties)
}

func TestToPropertyModels_Property_AppendsDottedName(t *testing.T) {
	properties := schemamapper.ToPropertyModels(nil, dsl.Schema{
		Type: "object",
		Properties: map[string]dsl.Schema{
			"x": {Type: "integer"},
		},
	})

	assert.Equal(t, map[string]model.Property{
		"$":   {Path: "$", Type: "object", Optional: false},
		"$.x": {Path: "$.x", Type: "integer", Optional: false},
	}, properties)
}

func TestToPropertyModels_ArrayItem_SuffixesBrackets(t *testing.T) {
	properties := schemamapper.ToPropertyModels(nil, dsl.Schema{
		Type: "object",
		Properties: map[string]dsl.Schema{
			"x": {Type: "array", Items: &dsl.Schema{Type: "string"}},
		},
	})

	assert.Equal(t, map[string]model.Property{
		"$":     {Path: "$", Type: "object", Optional: false},
		"$.x":   {Path: "$.x", Type: "array", Optional: false},
		"$.x[]": {Path: "$.x[]", Type: "string", Optional: false},
	}, properties)
}
