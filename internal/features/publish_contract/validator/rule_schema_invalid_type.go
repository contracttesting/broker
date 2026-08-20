package validator

import (
	"fmt"

	"github.com/contracttesting/broker/internal/dsl"
)

type schemaInvalidTypeRule struct{}

func (schemaInvalidTypeRule) Code() string { return "schema.invalid_type" }

func (schemaInvalidTypeRule) Validate(segment any, validationContext *ValidationContext) {
	schema, ok := segment.(dsl.Schema)
	if !ok {
		return
	}

	if validationContext.Depth.Exceeded() || schema.IsRef() || schema.IsArray() || schema.IsObject() || schema.IsPrimitive() {
		return
	}

	validationContext.AddViolation(fmt.Sprintf(
		"invalid schema type %q at %s (%s)",
		schema.Type,
		validationContext.Where,
		validationContext.Source,
	))
}
