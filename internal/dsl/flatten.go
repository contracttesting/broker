package dsl

// MAX_DEPTH is the budget every descent through a schema carries. Validation runs it
// over contracts whose refs may still cycle, so the walk has to stop on its own.
const MAX_DEPTH = 10

type FlatProperty struct {
	Type     string
	Optional bool
}

// FlattenSchema resolves a schema into the property paths it declares — `$` for the
// root, `$.x` for a property, `$.x[]` for an array item — following refs through the
// namespace. A node whose type is none of the supported ones keeps its declared type
// verbatim, so the caller can quote it back.
func FlattenSchema(schemas SchemasMap, root Schema) map[string]FlatProperty {
	properties := map[string]FlatProperty{}

	flattenSchema(schemas, properties, NewPropertyPath("$"), root, 0)

	return properties
}

func flattenSchema(
	schemas SchemasMap,
	properties map[string]FlatProperty,
	propertyPath PropertyPath,
	schema Schema,
	depth int,
) {
	if depth >= MAX_DEPTH {
		return
	}

	switch {
	case schema.IsObject():
		properties[propertyPath.String()] = FlatProperty{Type: "object", Optional: schema.Optional}

		for name, property := range schema.Properties {
			flattenSchema(schemas, properties, propertyPath.Append(name), property, depth+1)
		}

	case schema.IsArray():
		properties[propertyPath.String()] = FlatProperty{Type: "array", Optional: schema.Optional}

		if schema.Items == nil {
			return
		}

		flattenSchema(schemas, properties, propertyPath.AppendArray(), *schema.Items, depth+1)

	case schema.IsPrimitive():
		properties[propertyPath.String()] = FlatProperty{Type: schema.Type, Optional: schema.Optional}

	case schema.IsRef():
		// The site carries an optional of its own; the target knows nothing about it.
		// Either side declaring the node optional makes it optional.
		target := schemas[schema.Ref]
		target.Optional = target.Optional || schema.Optional

		flattenSchema(schemas, properties, propertyPath, target, depth+1)

	default:
		properties[propertyPath.String()] = FlatProperty{Type: schema.Type, Optional: schema.Optional}
	}
}
