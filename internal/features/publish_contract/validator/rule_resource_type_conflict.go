package validator

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/contracttesting/broker/internal/dsl"
)

type resourceTypeConflictRule struct {
	seen map[string]map[string]typedProperty
}

type typedProperty struct {
	propertyType string
	source       string
}

func (resourceTypeConflictRule) Code() string { return "resource.type_conflict" }

func (r *resourceTypeConflictRule) Validate(value any, contextualValidator *ContextualValidator) {
	resourceSchema, ok := value.(ResourceSchema)
	if !ok {
		return
	}

	// only consumed resources are merged, so only they can disagree with themselves; a
	// provider redeclaration is already rejected as a duplicate
	if !strings.HasPrefix(resourceSchema.Path, "consumes;") {
		return
	}

	schema, declared := contextualValidator.contractIndex.Schema(resourceSchema.SchemaName)
	if !declared {
		return
	}

	declaredTypes, tracked := r.seen[resourceSchema.Path]
	if !tracked {
		declaredTypes = map[string]typedProperty{}
		r.seen[resourceSchema.Path] = declaredTypes
	}

	flattened := dsl.FlattenSchema(contextualValidator.contractIndex.schemas, schema)

	for _, path := range slices.Sorted(maps.Keys(flattened)) {
		current := typedProperty{propertyType: flattened[path].Type, source: contextualValidator.source}

		previous, taken := declaredTypes[path]
		if !taken {
			declaredTypes[path] = current

			continue
		}

		if previous.propertyType == current.propertyType {
			continue
		}

		resourcePath := dsl.NewResourcePath(resourceSchema.Path)
		contextualValidator.addViolation(fmt.Sprintf(
			"conflicting property type for %s at %s: %s (%s) and %s (%s)",
			path,
			resourcePath.ToResource(nil).Describe(),
			previous.propertyType,
			previous.source,
			current.propertyType,
			current.source,
		))
	}
}
