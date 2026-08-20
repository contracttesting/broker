package validator

import (
	"strings"
)

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
