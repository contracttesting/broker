package validator

import (
	"maps"
	"slices"

	"github.com/contracttesting/broker/internal/dsl"
)

type endpointDuplicateRule struct{}

func (endpointDuplicateRule) Code() string { return "endpoint.duplicate" }

// Two spellings of the same path in one file would silently overwrite each other once
// normalized, so the first one in sorted order wins and the second is reported.
func (endpointDuplicateRule) CheckRest(r dsl.Rest, vctx ValidationContext) {
	seen := make(map[string]bool)

	for _, endpoint := range slices.Sorted(maps.Keys(r)) {
		normalized := dsl.NormalizeEndpoint(endpoint)
		if seen[normalized] {
			vctx.Errs.Addf("duplicate endpoint: %s declared twice in %s", normalized, vctx.Pos.Source)

			continue
		}

		seen[normalized] = true
	}
}
