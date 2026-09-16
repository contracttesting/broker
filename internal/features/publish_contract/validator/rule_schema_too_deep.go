package validator

import (
	"maps"
	"slices"
	"strconv"

	"github.com/contracttesting/broker/internal/features/publish_contract/dsl"
	"github.com/contracttesting/broker/internal/features/publish_contract/mapper/fragmentmapper"
	"github.com/contracttesting/broker/internal/features/publish_contract/mapper/schemamapper"
	"github.com/contracttesting/broker/internal/features/publish_contract/violation"
)

func schemasTooDeep(declarations fragmentmapper.Declarations) []violation.Violation {
	var violations []violation.Violation

	for _, declaration := range declarations.Schemas {
		if !exceedsDepth(declaration.Schema, declarations.Catalog, DepthCounter{}) {
			continue
		}

		violations = append(violations, violation.Violation{
			Code:   "schema.too_deep",
			Path:   schemaPath(declaration.Name),
			Source: declaration.Source,
			Details: map[string]string{
				"schema":   declaration.Name,
				"maxDepth": strconv.Itoa(schemamapper.MaxDepth),
			},
		})
	}

	return violations
}

func exceedsDepth(schema dsl.Schema, catalog dsl.SchemasMap, depth DepthCounter) bool {
	if depth.Exceeded() {
		return true
	}

	switch {
	case schema.IsRef():
		target, declared := catalog[schema.Ref]
		if !declared {
			return false
		}

		return exceedsDepth(target, catalog, depth.Deeper())

	case schema.IsArray():
		if schema.Items == nil {
			return false
		}

		return exceedsDepth(*schema.Items, catalog, depth.Deeper())

	case schema.IsObject():
		for _, name := range slices.Sorted(maps.Keys(schema.Properties)) {
			if exceedsDepth(schema.Properties[name], catalog, depth.Deeper()) {
				return true
			}
		}
	}

	return false
}
