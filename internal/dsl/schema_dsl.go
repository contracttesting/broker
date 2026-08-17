package dsl

import (
	"maps"
	"slices"
)

type SchemasMap map[string]Schema

// Validate walks every declared schema, referenced by an endpoint or not, each one the
// root of its own descent and of its own depth budget.
func (s SchemasMap) Validate(vctx ValidationContext) {
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
func (s Schema) Validate(vctx ValidationContext) {
	if vctx.Pos.Depth.Exceeded() {
		vctx.Errs.Addf(
			"schema %s is too deep with more than %d levels (%s)",
			vctx.Pos.Root,
			MAX_DEPTH,
			vctx.Pos.Source,
		)

		return
	}

	switch {
	case s.IsRef():
		target, declared := vctx.Index.Schema(s.Ref)
		if !declared {
			vctx.Errs.Addf(
				"unresolved schema name: %s referenced at %s (%s)",
				s.Ref,
				vctx.Pos.Path.String(),
				vctx.Pos.Source,
			)

			return
		}

		target.Validate(vctx.Deeper())

	case s.IsArray():
		if s.Items == nil {
			vctx.Errs.Addf(
				"array schema without items at %s (%s)",
				vctx.Pos.Path.String(),
				vctx.Pos.Source,
			)

			return
		}

		s.Items.Validate(vctx.Deeper().AtItems())

	case s.IsObject():
		for _, name := range slices.Sorted(maps.Keys(s.Properties)) {
			s.Properties[name].Validate(vctx.Deeper().AtProperty(name))
		}

	case !s.IsPrimitive():
		vctx.Errs.Addf(
			"invalid schema type %q at %s (%s)",
			s.Type,
			vctx.Pos.Path.String(),
			vctx.Pos.Source,
		)
	}
}
