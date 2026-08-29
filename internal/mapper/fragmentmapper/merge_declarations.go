package fragmentmapper

import (
	"fmt"

	"github.com/contracttesting/broker/internal/dsl"
	"github.com/contracttesting/broker/internal/model"
)

// resourceDeclaration is one leaf of one fragment: the resource it declares, keyed by
// the path that names it, and the file it came from, which the merge error quotes.
type resourceDeclaration struct {
	source       string
	resourcePath dsl.ResourcePath
	resource     model.UploadedResource
}

// mergeDeclarationsByResourcePath folds every declaration of the same resource path
// into one, in first-appearance order. A consumer declared by several modules is the
// union of what they declare; a provider is declared once, and a second declaration
// is an error naming both files.
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

// unionResourceModels joins two declarations of the same resource. Identity comes from
// the first (equal by construction); the properties are read the way the compatibility
// check will, by the resource's interaction.
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

// unionRequestProperties: the app only sends what every module sends, so a path
// missing from — or optional in — one declaration is optional for all.
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

// unionResponseProperties: the strongest reader wins, and a module that never mentions
// the path has no say in it.
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
