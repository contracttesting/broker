package dsl

import (
	"maps"
	"slices"
)

// Normalize rewrites every endpoint of every fragment to its normalized spelling, in
// place, so validation and build only ever see one spelling of a path. It is the only
// normalization the publish does: everything else that feeds a hash is rejected.
func Normalize(fragments []Fragment) []error {
	errs := &ErrorList{}

	for _, fragment := range sortedBySource(fragments) {
		normalizeRest(fragment.Contract.Provides.Rest, fragment.Source, errs)

		for _, service := range slices.Sorted(maps.Keys(fragment.Contract.ConsumesServices)) {
			normalizeRest(fragment.Contract.ConsumesServices[service].Rest, fragment.Source, errs)
		}
	}

	return errs.All()
}

// normalizeRest renames the endpoints of one map. Two spellings of the same path in the
// same map would silently overwrite each other, so the first one in sorted order stays
// and the second is reported.
func normalizeRest(rest Rest, source string, errs *ErrorList) {
	for _, endpoint := range slices.Sorted(maps.Keys(rest)) {
		normalized := normalizeEndpoint(endpoint)
		if normalized == endpoint {
			continue
		}

		if _, taken := rest[normalized]; taken {
			errs.Addf("duplicate endpoint: %s declared twice in %s", normalized, source)
			delete(rest, endpoint)

			continue
		}

		rest[normalized] = rest[endpoint]
		delete(rest, endpoint)
	}
}
