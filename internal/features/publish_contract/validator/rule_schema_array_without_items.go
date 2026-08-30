package validator

import (
	"fmt"

	"github.com/contracttesting/broker/internal/features/publish_contract/dsl"
)

type schemaArrayWithoutItemsRule struct{}

func (schemaArrayWithoutItemsRule) Code() string { return "schema.array_without_items" }

func (schemaArrayWithoutItemsRule) Validate(value any, contextualValidator *ContextualValidator) {
	schema, ok := value.(dsl.Schema)
	if !ok {
		return
	}

	if contextualValidator.depth.Exceeded() || schema.IsRef() || !schema.IsArray() || schema.Items != nil {
		return
	}

	contextualValidator.addViolation(fmt.Sprintf(
		"array schema without items at %s (%s)",
		contextualValidator.where,
		contextualValidator.source,
	))
}
