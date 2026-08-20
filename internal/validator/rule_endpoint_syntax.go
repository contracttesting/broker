package validator

import (
	"fmt"

	"github.com/contracttesting/broker/internal/dsl"
)

type endpointSyntaxRule struct{}

func (endpointSyntaxRule) Code() string { return "endpoint.syntax" }

func (endpointSyntaxRule) Validate(segment any, validationContext *ValidationContext) {
	endpoint, ok := segment.(string)
	if !ok {
		return
	}

	if reason := endpointViolation(dsl.NormalizeEndpoint(endpoint)); reason != "" {
		validationContext.AddViolation(fmt.Sprintf("invalid endpoint %q: %s (%s)", endpoint, reason, validationContext.Source))
	}
}
