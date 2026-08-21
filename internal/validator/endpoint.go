package validator

import (
	"strings"
)

// endpointViolation says why an endpoint is invalid, or "" when it is valid. It expects
// a normalized endpoint: a trailing slash past the root is a malformed path here, not a
// spelling to be trimmed.
func endpointViolation(endpoint string) string {
	if !strings.HasPrefix(endpoint, "/") {
		return "malformed path"
	}

	if endpoint == "/" {
		return ""
	}

	for _, segment := range strings.Split(endpoint[1:], "/") {
		if segment == "" || strings.Contains(segment, ";") {
			return "malformed path"
		}

		if segment == "*" {
			continue
		}

		if strings.ContainsAny(segment, "*{}") {
			return "dynamic path segments must use *"
		}
	}

	return ""
}
