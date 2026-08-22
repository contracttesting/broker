package validator

import "fmt"

type schemaUnresolvedNameRule struct{}

func (schemaUnresolvedNameRule) Code() string { return "schema.unresolved_name" }

func (schemaUnresolvedNameRule) Validate(value any, contextualValidator *ContextualValidator) {
	name, ok := value.(string)
	if !ok {
		return
	}

	if _, declared := contextualValidator.contractIndex.Schema(name); !declared {
		contextualValidator.addViolation(fmt.Sprintf(
			"unresolved schema name: %s referenced at %s (%s)",
			name,
			contextualValidator.where,
			contextualValidator.source,
		))
	}
}
