package validator

import (
	"fmt"

	"github.com/contracttesting/broker/internal/dsl"
	"github.com/contracttesting/broker/internal/mapper/resourcepathmapper"
)

type resourceDuplicateRule struct {
	seen map[string]string
}

func (resourceDuplicateRule) Code() string { return "resource.duplicate" }

func (r *resourceDuplicateRule) Validate(value any, contextualValidator *ContextualValidator) {
	path, ok := value.(string)
	if !ok {
		return
	}

	// only a provider redeclaration is an error: consumers merge by union in the mapper
	resourcePath := dsl.NewResourcePath(path)
	if !resourcePath.IsProvider() {
		return
	}

	if declaredIn, taken := r.seen[path]; taken {
		resource := resourcepathmapper.ToResourceModel(resourcePath, nil)
		contextualValidator.addViolation(r.message(
			resource.Describe(),
			declaredIn,
			contextualValidator.source,
		))

		return
	}

	r.seen[path] = contextualValidator.source
}

func (resourceDuplicateRule) message(resource, declaredIn, source string) string {
	if declaredIn == source {
		return fmt.Sprintf("duplicate resource: %s declared twice in %s", resource, source)
	}

	return fmt.Sprintf("duplicate resource: %s declared in %s and %s", resource, declaredIn, source)
}
