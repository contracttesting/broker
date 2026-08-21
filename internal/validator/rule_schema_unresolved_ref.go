package validator

import (
	"github.com/contracttesting/broker/internal/dsl"
)

type schemaUnresolvedRefRule struct{}

func (schemaUnresolvedRefRule) Code() string { return "schema.unresolved-ref" }

func (schemaUnresolvedRefRule) CheckSchema(s dsl.Schema, vctx ValidationContext) {
	if vctx.Pos.Depth.Exceeded() || !s.IsRef() {
		return
	}

	if _, declared := vctx.Index.Schema(s.Ref); !declared {
		vctx.Errs.Addf(
			"unresolved schema name: %s referenced at %s (%s)",
			s.Ref,
			vctx.Pos.Path.String(),
			vctx.Pos.Source,
		)
	}
}
