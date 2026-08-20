package validator

import "fmt"

type schemaUnresolvedNameRule struct{}

func (schemaUnresolvedNameRule) Code() string { return "schema.unresolved_name" }

func (schemaUnresolvedNameRule) Validate(segment any, validationContext *ValidationContext) {
	name, ok := segment.(string)
	if !ok {
		return
	}

	if _, declared := validationContext.ContractIndex.Schema(name); !declared {
		validationContext.AddViolation(fmt.Sprintf(
			"unresolved schema name: %s referenced at %s (%s)",
			name,
			validationContext.Where,
			validationContext.Source,
		))
	}
}
