package dsl

import (
	"maps"
	"slices"
)

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
		normalized := NormalizeEndpoint(endpoint)
		if normalized == endpoint {
			continue
		}

		rest[normalized] = rest[endpoint]
		delete(rest, endpoint)
	}
}
