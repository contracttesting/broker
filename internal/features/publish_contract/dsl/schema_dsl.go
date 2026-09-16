package dsl

type SchemasMap map[string]Schema

type Schema struct {
	Type        string
	Description string
	Properties  map[string]Schema
	Items       *Schema
	Ref         string
	Optional    bool
}

func SchemaFromDocument(document Document) Schema {
	schema := Schema{
		Type:        document.Text("type"),
		Description: document.Text("description"),
		Ref:         document.Text("ref"),
		Optional:    document.Flag("optional"),
	}

	if properties, written := document["properties"].(map[string]any); written {
		schema.Properties = make(map[string]Schema, len(properties))

		for name := range properties {
			schema.Properties[name] = SchemaFromDocument(Document(properties).Mapping(name))
		}
	}

	if items, written := document["items"].(map[string]any); written {
		itemsSchema := SchemaFromDocument(items)
		schema.Items = &itemsSchema
	}

	return schema
}

func (s *Schema) IsObject() bool {
	if s.Type != "" {
		return s.Type == "object"
	}

	return s.Properties != nil
}

func (s *Schema) IsArray() bool {
	if s.Type != "" {
		return s.Type == "array"
	}

	return s.Items != nil
}

func (s *Schema) IsPrimitive() bool {
	if s.Type != "" {
		return isPrimitiveType(s.Type)
	}

	return false
}

func isPrimitiveType(schemaType string) bool {
	return schemaType == "string" ||
		schemaType == "integer" ||
		schemaType == "float" ||
		schemaType == "boolean"
}

func (s *Schema) IsRef() bool {
	if s.Type != "" || s.Properties != nil || s.Items != nil {
		return false
	}

	return s.Ref != ""
}
