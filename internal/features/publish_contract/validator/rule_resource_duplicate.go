package validator

import (
	"fmt"
	"strings"

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

	// a consumed resource declared twice is merged by union at build time; a provider
	// has no second source of truth for the same interaction
	if !strings.HasPrefix(resource, "provides;") {
		return
	}

	if declaredIn, taken := r.seen[resource]; taken {
		resourcePath := dsl.NewResourcePath(resource)
		contextualValidator.addViolation(r.message(
			resourcePath.ToResource(nil).Describe(),
			declaredIn,
			contextualValidator.source,
		))

		return
	}

	r.seen[resource] = contextualValidator.source
}

func (resourceDuplicateRule) message(resource, declaredIn, source string) string {
	if declaredIn == source {
		return fmt.Sprintf("duplicate resource: %s declared twice in %s", resource, source)
	}

	return fmt.Sprintf("duplicate resource: %s declared in %s and %s", resource, declaredIn, source)
}
