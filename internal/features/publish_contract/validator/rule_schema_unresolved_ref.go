package validator

import (
	"fmt"

	"github.com/contracttesting/broker/internal/dsl"
)

type schemaUnresolvedRefRule struct{}

func (schemaUnresolvedRefRule) Code() string { return "schema.unresolved_ref" }

func (schemaUnresolvedRefRule) Validate(value any, contextualValidator *ContextualValidator) {
	schema, ok := value.(dsl.Schema)
	if !ok {
		return
	}

	if contextualValidator.depth.Exceeded() || !schema.IsRef() {
		return
	}

	if _, declared := contextualValidator.contractIndex.Schema(schema.Ref); !declared {
		contextualValidator.addViolation(fmt.Sprintf(
			"unresolved schema name: %s referenced at %s (%s)",
			schema.Ref,
			contextualValidator.where,
			contextualValidator.source,
		))
	}
}
