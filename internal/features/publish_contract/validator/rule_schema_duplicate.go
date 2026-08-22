package validator

import (
	"fmt"
)

type schemaDuplicateRule struct {
	seen map[string]string
}

func (schemaDuplicateRule) Code() string { return "schema.duplicate" }

func (r *schemaDuplicateRule) Validate(value any, contextualValidator *ContextualValidator) {
	name, ok := value.(string)
	if !ok {
		return
	}

	if declaredIn, taken := r.seen[name]; taken {
		contextualValidator.addViolation(fmt.Sprintf("duplicate schema: %s declared in %s and %s", name, declaredIn, contextualValidator.source))

		return
	}

	r.seen[name] = contextualValidator.source
}
