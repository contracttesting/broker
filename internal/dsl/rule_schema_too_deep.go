package dsl

type schemaTooDeepRule struct{}

func (schemaTooDeepRule) Code() string { return "schema.too-deep" }

func (schemaTooDeepRule) CheckSchema(_ Schema, vctx ValidationContext) {
	if !vctx.Pos.Depth.Exceeded() {
		return
	}

	vctx.Errs.Addf(
		"schema %s is too deep with more than %d levels (%s)",
		vctx.Pos.Root,
		MAX_DEPTH,
		vctx.Pos.Source,
	)
}
