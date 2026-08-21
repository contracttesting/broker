package validator

// The schema itself is walked once, from its declaration, so nothing descends here.
type schemaUnresolvedNameRule struct{}

func (schemaUnresolvedNameRule) Code() string { return "schema.unresolved-name" }

func (schemaUnresolvedNameRule) CheckSchemaName(name string, vctx ValidationContext) {
	if _, declared := vctx.Index.Schema(name); !declared {
		vctx.Errs.Addf(
			"unresolved schema name: %s referenced at %s (%s)",
			name,
			vctx.Pos.Where,
			vctx.Pos.Source,
		)
	}
}
