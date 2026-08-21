package validator

import (
	"github.com/contracttesting/broker/internal/dsl"
)

type schemaInvalidTypeRule struct{}

func (schemaInvalidTypeRule) Code() string { return "schema.invalid-type" }

func (schemaInvalidTypeRule) CheckSchema(s dsl.Schema, vctx ValidationContext) {
	if vctx.Pos.Depth.Exceeded() || s.IsRef() || s.IsArray() || s.IsObject() || s.IsPrimitive() {
		return
	}

	vctx.Errs.Addf(
		"invalid schema type %q at %s (%s)",
		s.Type,
		vctx.Pos.Path.String(),
		vctx.Pos.Source,
	)
}
