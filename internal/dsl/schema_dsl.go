package dsl

import (
	"maps"
	"slices"
)

type SchemasMap map[string]Schema

// Validate walks every declared schema, referenced by an endpoint or not, each one the
// root of its own descent and of its own depth budget.
func (s SchemasMap) Validate(vctx ValidationContext) {
	vctx.checkSchemas(s)

	for _, name := range slices.Sorted(maps.Keys(s)) {
		s[name].Validate(vctx.AtSchema(name))
	}
}

type Schema struct {
	Type        string            `json:"type,omitzero"`
	Description string            `json:"description,omitzero"`
	Properties  map[string]Schema `json:"properties,omitzero"`
	Items       *Schema           `json:"items,omitzero"`
	Ref         string            `json:"ref,omitzero"`
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

// Validate descends the schema, following its refs through the namespace. Following a
// ref is what catches a cycle: the descent only ends when the depth budget runs out.
// The messages come from the rules; the gates that stop a dead-end descent are the
// walker's own.
func (s Schema) Validate(vctx ValidationContext) {
	vctx.checkSchema(s)

	if vctx.Pos.Depth.Exceeded() {
		return
	}

	switch {
	case s.IsRef():
		target, declared := vctx.Index.Schema(s.Ref)
		if !declared {
			return
		}

		target.Validate(vctx.Deeper())

	case s.IsArray():
		if s.Items == nil {
			return
		}

		s.Items.Validate(vctx.Deeper().AtItems())

	case s.IsObject():
		for _, name := range slices.Sorted(maps.Keys(s.Properties)) {
			s.Properties[name].Validate(vctx.Deeper().AtProperty(name))
		}
	}
}
