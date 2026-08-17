package dsl

import (
	"strings"
)

// NormalizeEndpoint is the single source of truth for the one spelling rewrite the DSL
// tolerates: a trailing slash past the root carries no meaning and is trimmed.
func NormalizeEndpoint(endpoint string) string {
	if len(endpoint) > 1 && strings.HasSuffix(endpoint, "/") {
		return strings.TrimSuffix(endpoint, "/")
	}

	return endpoint
}
