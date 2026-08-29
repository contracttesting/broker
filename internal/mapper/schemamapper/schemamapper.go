package schemamapper

import (
	"github.com/contracttesting/broker/internal/dsl"
	"github.com/contracttesting/broker/internal/model"
)

// MaxDepth bounds the schema descent so cyclic refs still terminate.
const MaxDepth = 10

// ToPropertyModels resolves a schema into its property models keyed by $-rooted path.
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
		target := schemas[schema.Ref]
		target.Optional = target.Optional || schema.Optional

		propertyModelsFromSchema(schemas, properties, path, target, depth+1)

	default:
		properties[path.String()] = model.NewProperty(path.String(), schema.Type, schema.Optional)
	}
}
