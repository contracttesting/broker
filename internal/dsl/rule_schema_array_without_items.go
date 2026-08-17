package dsl

type schemaArrayWithoutItemsRule struct{}

func (schemaArrayWithoutItemsRule) Code() string { return "schema.array-without-items" }

func (schemaArrayWithoutItemsRule) CheckSchema(s Schema, vctx ValidationContext) {
	if vctx.Pos.Depth.Exceeded() || s.IsRef() || !s.IsArray() || s.Items != nil {
		return
	}

	vctx.Errs.Addf(
		"array schema without items at %s (%s)",
		vctx.Pos.Path.String(),
		vctx.Pos.Source,
	)
}
