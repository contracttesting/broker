package validator

import (
	"fmt"

	"github.com/contracttesting/broker/internal/dsl"
)

type resourceDuplicateRule struct {
	seen map[string]string
}

func (resourceDuplicateRule) Code() string { return "resource.duplicate" }

func (r *resourceDuplicateRule) Validate(value any, contextualValidator *ContextualValidator) {
	resource, ok := value.(string)
	if !ok {
		return
	}

	if declaredIn, taken := r.seen[resource]; taken {
		resourcePath := dsl.NewResourcePath(resource)
		contextualValidator.addViolation(fmt.Sprintf(
			"duplicate resource: %s declared in %s and %s",
			resourcePath.ToResource(nil).Describe(),
			declaredIn,
			contextualValidator.source,
		))

		return
	}

	r.seen[resource] = contextualValidator.source
}
