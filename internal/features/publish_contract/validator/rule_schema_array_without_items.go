package validator

import (
	"fmt"

	"github.com/contracttesting/broker/internal/dsl"
)

type schemaArrayWithoutItemsRule struct{}

func (schemaArrayWithoutItemsRule) Code() string { return "schema.array_without_items" }

func (schemaArrayWithoutItemsRule) Validate(segment any, validationContext *ValidationContext) {
	schema, ok := segment.(dsl.Schema)
	if !ok {
		return
	}

	if validationContext.Depth.Exceeded() || schema.IsRef() || !schema.IsArray() || schema.Items != nil {
		return
	}

	validationContext.AddViolation(fmt.Sprintf(
		"array schema without items at %s (%s)",
		validationContext.Where,
		validationContext.Source,
	))
}
