package validator

import (
	"fmt"
	"maps"
	"slices"

	"github.com/contracttesting/broker/internal/dsl"
)

type schemaDuplicateRule struct {
	seen map[string]string
}

func (schemaDuplicateRule) Code() string { return "schema.duplicate" }

func (schemaDuplicateRule) Fresh() StatefulRule {
	return &schemaDuplicateRule{seen: map[string]string{}}
}

func (r *schemaDuplicateRule) Validate(segment any, validationContext *ValidationContext) {
	schemas, ok := segment.(dsl.SchemasMap)
	if !ok {
		return
	}

	for _, name := range slices.Sorted(maps.Keys(schemas)) {
		if declaredIn, taken := r.seen[name]; taken {
			validationContext.AddViolation(fmt.Sprintf("duplicate schema: %s declared in %s and %s", name, declaredIn, validationContext.Source))

			continue
		}

		r.seen[name] = validationContext.Source
	}
}
