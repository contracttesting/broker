package validator

import (
	"fmt"
	"maps"
	"slices"

	"github.com/contracttesting/broker/internal/features/publish_contract/dsl"
	"github.com/contracttesting/broker/internal/features/publish_contract/mapper/resourcepathmapper"
	"github.com/contracttesting/broker/internal/features/publish_contract/mapper/schemamapper"
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

	// only consumed resources merge, so only they can disagree with themselves
	resourcePath := dsl.NewResourcePath(resourceSchema.Path)
	if !resourcePath.IsConsumer() {
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

	properties := schemamapper.ToPropertyModels(contextualValidator.contractIndex.schemas, schema)

	for _, path := range slices.Sorted(maps.Keys(properties)) {
		current := typedProperty{propertyType: properties[path].Type, source: contextualValidator.source}

		previous, taken := declaredTypes[path]
		if !taken {
			declaredTypes[path] = current

			continue
		}

		if previous.propertyType == current.propertyType {
			continue
		}

		resource := resourcepathmapper.ToResourceModel(resourcePath, nil)
		contextualValidator.addViolation(fmt.Sprintf(
			"conflicting property type for %s at %s: %s (%s) and %s (%s)",
			path,
			resource.Describe(),
			previous.propertyType,
			previous.source,
			current.propertyType,
			current.source,
		))
	}
}
