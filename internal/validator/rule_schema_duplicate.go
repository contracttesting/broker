package validator

import (
	"maps"
	"slices"

	"github.com/contracttesting/broker/internal/dsl"
)

type schemaDuplicateRule struct {
	seen map[string]string
}

func (schemaDuplicateRule) Code() string { return "schema.duplicate" }

func (schemaDuplicateRule) Fresh() Rule { return &schemaDuplicateRule{seen: map[string]string{}} }

func (r *schemaDuplicateRule) CheckSchemas(s dsl.SchemasMap, vctx ValidationContext) {
	for _, name := range slices.Sorted(maps.Keys(s)) {
		if declaredIn, taken := r.seen[name]; taken {
			vctx.Errs.Addf("duplicate schema: %s declared in %s and %s", name, declaredIn, vctx.Pos.Source)

			continue
		}

		r.seen[name] = vctx.Pos.Source
	}
}
