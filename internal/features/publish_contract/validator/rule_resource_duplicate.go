package validator

import (
	"fmt"

	"github.com/contracttesting/broker/internal/dsl"
)

type resourceDuplicateRule struct {
	seen map[string]string
}

func (resourceDuplicateRule) Code() string { return "resource.duplicate" }

func (resourceDuplicateRule) Fresh() StatefulRule {
	return &resourceDuplicateRule{seen: map[string]string{}}
}

func (r *resourceDuplicateRule) Validate(value any, validationContext *ValidationContext) {
	resource, ok := value.(string)
	if !ok {
		return
	}

	if declaredIn, taken := r.seen[resource]; taken {
		resourcePath := dsl.NewResourcePath(resource)
		validationContext.AddViolation(fmt.Sprintf(
			"duplicate resource: %s declared in %s and %s",
			resourcePath.ToResource(nil).Describe(),
			declaredIn,
			validationContext.Source,
		))

		return
	}

	r.seen[resource] = validationContext.Source
}
