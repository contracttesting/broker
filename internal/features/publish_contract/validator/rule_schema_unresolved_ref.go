package validator

import (
	"fmt"

	"github.com/contracttesting/broker/internal/dsl"
)

type schemaUnresolvedRefRule struct{}

func (schemaUnresolvedRefRule) Code() string { return "schema.unresolved_ref" }

func (schemaUnresolvedRefRule) Validate(segment any, validationContext *ValidationContext) {
	schema, ok := segment.(dsl.Schema)
	if !ok {
		return
	}

	if validationContext.Depth.Exceeded() || !schema.IsRef() {
		return
	}

	if _, declared := validationContext.ContractIndex.Schema(schema.Ref); !declared {
		validationContext.AddViolation(fmt.Sprintf(
			"unresolved schema name: %s referenced at %s (%s)",
			schema.Ref,
			validationContext.Where,
			validationContext.Source,
		))
	}
}
