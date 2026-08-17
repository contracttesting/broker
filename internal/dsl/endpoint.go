package dsl

import (
	"fmt"
	"strings"
)

func normalizeEndpoint(endpoint string) string {
	if len(endpoint) > 1 && strings.HasSuffix(endpoint, "/") {
		return strings.TrimSuffix(endpoint, "/")
	}

	return endpoint
}

// validateEndpoint expects an endpoint that Normalize already rewrote: a trailing slash
// past the root is a malformed path here, not a spelling to be trimmed.
func validateEndpoint(endpoint string) error {
	if !strings.HasPrefix(endpoint, "/") {
		return fmt.Errorf("invalid endpoint %q: malformed path", endpoint)
	}

	if endpoint == "/" {
		return nil
	}

	for _, segment := range strings.Split(endpoint[1:], "/") {
		if segment == "" || strings.Contains(segment, ";") {
			return fmt.Errorf("invalid endpoint %q: malformed path", endpoint)
		}

		if segment == "*" {
			continue
		}

		if strings.ContainsAny(segment, "*{}") {
			return fmt.Errorf("invalid endpoint %q: dynamic path segments must use *", endpoint)
		}
	}

	return nil
}
