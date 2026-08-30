package validator

import (
	"fmt"

	"github.com/contracttesting/broker/internal/features/publish_contract/dsl"
	"github.com/contracttesting/broker/internal/validations"
)

type endpointSyntaxRule struct{}

func (endpointSyntaxRule) Code() string { return "endpoint.syntax" }

func (endpointSyntaxRule) Validate(value any, contextualValidator *ContextualValidator) {
	endpoint, ok := value.(string)
	if !ok {
		return
	}

	if err := validations.Endpoint(dsl.NormalizeEndpoint(endpoint)); err != nil {
		contextualValidator.addViolation(fmt.Sprintf("invalid endpoint %q: %s (%s)", endpoint, err, contextualValidator.source))
	}
}
