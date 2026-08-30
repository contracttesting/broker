package fragmentmapper

import (
	"fmt"

	"github.com/contracttesting/broker/internal/features/publish_contract/dsl"
	"github.com/contracttesting/broker/internal/model"
)

// resourceDeclaration is the resource one fragment leaf declares, with its path key and source file.
type resourceDeclaration struct {
	source       string
	resourcePath dsl.ResourcePath
	resource     model.UploadedResource
}

// mergeDeclarationsByResourcePath folds the declarations of each resource path into one, in first-appearance order.
func mergeDeclarationsByResourcePath(declarations []resourceDeclaration) ([]resourceDeclaration, error) {
	merged := make([]resourceDeclaration, 0, len(declarations))
	indexByPath := make(map[string]int, len(declarations))

	for _, declaration := range declarations {
		key := declaration.resourcePath.String()

		index, seen := indexByPath[key]
		if !seen {
			indexByPath[key] = len(merged)
			merged = append(merged, declaration)

			continue
		}

		first := merged[index]

		if declaration.resource.IsProvider() {
			return nil, fmt.Errorf(
				"resource already added: %s from %s and %s",
				declaration.resource.Describe(),
				first.source,
				declaration.source,
			)
		}

		merged[index].resource = unionResourceModels(first.resource, declaration.resource)
	}

	return merged, nil
}

// unionResourceModels joins two declarations of the same resource by its interaction.
func unionResourceModels(a, b model.UploadedResource) model.UploadedResource {
	union := a

	switch a.Interaction {
	case model.RestRequest:
		union.Properties = unionRequestProperties(a.Properties, b.Properties)
	case model.RestResponse:
		union.Properties = unionResponseProperties(a.Properties, b.Properties)
	}

	return union
}

// unionRequestProperties merges what two senders send: a path is required only if both require it.
func unionRequestProperties(a, b map[string]model.Property) map[string]model.Property {
	union := make(map[string]model.Property, len(a)+len(b))

	for path, property := range a {
		other, declared := b[path]
		property.Optional = !declared || property.Optional || other.Optional
		union[path] = property
	}

	for path, property := range b {
		if _, declared := a[path]; declared {
			continue
		}

		property.Optional = true
		union[path] = property
	}

	return union
}

// unionResponseProperties merges what two readers need: a path is optional only if every reader that mentions it allows it.
func unionResponseProperties(a, b map[string]model.Property) map[string]model.Property {
	union := make(map[string]model.Property, len(a)+len(b))

	for path, property := range a {
		if other, declared := b[path]; declared {
			property.Optional = property.Optional && other.Optional
		}

		union[path] = property
	}

	for path, property := range b {
		if _, declared := a[path]; declared {
			continue
		}

		union[path] = property
	}

	return union
}
