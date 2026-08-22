package validator

import (
	"fmt"

	"github.com/contracttesting/broker/internal/dsl"
)

type schemaInvalidTypeRule struct{}

func (schemaInvalidTypeRule) Code() string { return "schema.invalid_type" }

func (schemaInvalidTypeRule) Validate(value any, contextualValidator *ContextualValidator) {
	schema, ok := value.(dsl.Schema)
	if !ok {
		return
	}

	if contextualValidator.depth.Exceeded() || schema.IsRef() || schema.IsArray() || schema.IsObject() || schema.IsPrimitive() {
		return
	}

	contextualValidator.addViolation(fmt.Sprintf(
		"invalid schema type %q at %s (%s)",
		schema.Type,
		contextualValidator.where,
		contextualValidator.source,
	))
}
