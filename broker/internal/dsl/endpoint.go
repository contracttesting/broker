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

func validateEndpoint(endpoint string) error {
	normalized := normalizeEndpoint(endpoint)

	if !strings.HasPrefix(normalized, "/") {
		return fmt.Errorf("invalid endpoint %q: malformed path", endpoint)
	}

	if normalized == "/" {
		return nil
	}

	for _, segment := range strings.Split(normalized[1:], "/") {
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
