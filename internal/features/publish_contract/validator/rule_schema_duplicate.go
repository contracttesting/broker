package validator

import (
	"fmt"
)

type schemaDuplicateRule struct {
	seen map[string]string
}

func (schemaDuplicateRule) Code() string { return "schema.duplicate" }

func (schemaDuplicateRule) Fresh() StatefulRule {
	return &schemaDuplicateRule{seen: map[string]string{}}
}

func (r *schemaDuplicateRule) Validate(value any, validationContext *ValidationContext) {
	name, ok := value.(string)
	if !ok {
		return
	}

	if declaredIn, taken := r.seen[name]; taken {
		validationContext.AddViolation(fmt.Sprintf("duplicate schema: %s declared in %s and %s", name, declaredIn, validationContext.Source))

		return
	}

	r.seen[name] = validationContext.Source
}
