package validator

import (
	"fmt"
	"maps"
	"slices"

	"github.com/contracttesting/broker/internal/dsl"
)

type endpointDuplicateRule struct{}

func (endpointDuplicateRule) Code() string { return "endpoint.duplicate" }

func (endpointDuplicateRule) Validate(segment any, validationContext *ValidationContext) {
	rest, ok := segment.(dsl.Rest)
	if !ok {
		return
	}

	seen := make(map[string]bool)

	for _, endpoint := range slices.Sorted(maps.Keys(rest)) {
		normalized := dsl.NormalizeEndpoint(endpoint)
		if seen[normalized] {
			validationContext.AddViolation(fmt.Sprintf("duplicate endpoint: %s declared twice in %s", normalized, validationContext.Source))

			continue
		}

		seen[normalized] = true
	}
}
