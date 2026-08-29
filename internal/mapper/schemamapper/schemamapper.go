package schemamapper

import (
	"github.com/contracttesting/broker/internal/dsl"
	"github.com/contracttesting/broker/internal/model"
)

// MaxDepth is the budget every descent through a schema carries. Validation runs it
// over contracts whose refs may still cycle, so the walk has to stop on its own.
const MaxDepth = 10

// ToPropertyModels resolves a schema into the property models it declares — `$` for
// the root, `$.x` for a property, `$.x[]` for an array item — following refs through
// the namespace. A node whose type is none of the supported ones keeps its declared
// type verbatim, so the caller can quote it back.
func ToPropertyModels(schemas dsl.SchemasMap, root dsl.Schema) map[string]model.Property {
	properties := map[string]model.Property{}

	propertyModelsFromSchema(schemas, properties, propertyPath("$"), root, 0)

	return properties
}

func propertyModelsFromSchema(
	schemas dsl.SchemasMap,
	properties map[string]model.Property,
	path propertyPath,
	schema dsl.Schema,
	depth int,
) {
	if depth >= MaxDepth {
		return
	}

	switch {
	case schema.IsObject():
		properties[path.String()] = model.NewProperty(path.String(), "object", schema.Optional)

		for name, property := range schema.Properties {
			propertyModelsFromSchema(schemas, properties, path.Append(name), property, depth+1)
		}

	case schema.IsArray():
		properties[path.String()] = model.NewProperty(path.String(), "array", schema.Optional)

		if schema.Items == nil {
			return
		}

		propertyModelsFromSchema(schemas, properties, path.AppendArray(), *schema.Items, depth+1)

	case schema.IsPrimitive():
		properties[path.String()] = model.NewProperty(path.String(), schema.Type, schema.Optional)

	case schema.IsRef():
		// The site carries an optional of its own; the target knows nothing about it.
		// Either side declaring the node optional makes it optional.
		target := schemas[schema.Ref]
		target.Optional = target.Optional || schema.Optional

		propertyModelsFromSchema(schemas, properties, path, target, depth+1)

	default:
		properties[path.String()] = model.NewProperty(path.String(), schema.Type, schema.Optional)
	}
}
