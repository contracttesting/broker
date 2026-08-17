package dsl

type resourceDuplicateRule struct {
	seen map[string]string
}

func (resourceDuplicateRule) Code() string { return "resource.duplicate" }

func (resourceDuplicateRule) Fresh() Rule { return &resourceDuplicateRule{seen: map[string]string{}} }

func (r *resourceDuplicateRule) CheckResource(resource string, vctx ValidationContext) {
	if declaredIn, taken := r.seen[resource]; taken {
		resourcePath := NewResourcePath(resource)
		vctx.Errs.Addf(
			"duplicate resource: %s declared in %s and %s",
			resourcePath.ToResource(nil).Describe(),
			declaredIn,
			vctx.Pos.Source,
		)

		return
	}

	r.seen[resource] = vctx.Pos.Source
}
