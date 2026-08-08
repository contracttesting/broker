package dsl

type SchemasMap map[string]Schema

type Schema struct {
	Type        string            `json:"type,omitzero"`
	Description string            `json:"description,omitzero"`
	Properties  map[string]Schema `json:"properties,omitzero"`
	Items       *Schema           `json:"items,omitzero"`
	Ref         string            `json:"$ref,omitzero"`
	Optional    bool              `json:"optional,omitzero"`
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
		return s.Type == "string" ||
			s.Type == "integer" ||
			s.Type == "float" ||
			s.Type == "number" ||
			s.Type == "boolean"
	}

	return false
}

func (s *Schema) IsRef() bool {
	if s.Type != "" || s.Properties != nil || s.Items != nil {
		return false
	}

	return s.Ref != ""
}
