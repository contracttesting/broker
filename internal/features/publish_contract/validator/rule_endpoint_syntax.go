package validator

import (
	"fmt"

	"github.com/contracttesting/broker/internal/common"
	"github.com/contracttesting/broker/internal/validations"
)

type endpointSyntaxRule struct{}

func (endpointSyntaxRule) Code() string { return "endpoint.syntax" }

func (endpointSyntaxRule) Validate(value any, contextualValidator *ContextualValidator) {
	endpoint, ok := value.(string)
	if !ok {
		return
	}

	if err := validations.Endpoint(common.NormalizeEndpoint(endpoint)); err != nil {
		contextualValidator.addViolation(fmt.Sprintf("invalid endpoint %q: %s (%s)", endpoint, err, contextualValidator.source))
	}
}
