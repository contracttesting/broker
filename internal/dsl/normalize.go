package dsl

import (
	"maps"
	"slices"
)

// Normalize rewrites every endpoint of every fragment to its normalized spelling, in
// place, so the build only ever sees one spelling of a path. It runs after validation
// passed, so it never meets a collision: it is a pure transformation. It is the only
// normalization the publish does: everything else that feeds a hash is rejected.
func Normalize(fragments []Fragment) {
	for _, fragment := range fragments {
		normalizeRest(fragment.Contract.Provides.Rest)

		for _, service := range fragment.Contract.ConsumesServices {
			normalizeRest(service.Rest)
		}
	}
}

func normalizeRest(rest Rest) {
	for _, endpoint := range slices.Sorted(maps.Keys(rest)) {
		normalized := normalizeEndpoint(endpoint)
		if normalized == endpoint {
			continue
		}

		rest[normalized] = rest[endpoint]
		delete(rest, endpoint)
	}
}
