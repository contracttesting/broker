package validator

import (
	"fmt"

	"github.com/contracttesting/broker/internal/common"
	"github.com/contracttesting/broker/internal/validations"
)

type endpointSyntaxRule struct{}

func (endpointSyntaxRule) Code() string { return "endpoint.syntax" }

func (endpointSyntaxRule) Validate(value any, validationContext *ValidationContext) {
	endpoint, ok := value.(string)
	if !ok {
		return
	}

	if err := validations.Endpoint(common.NormalizeEndpoint(endpoint)); err != nil {
		validationContext.AddViolation(fmt.Sprintf("invalid endpoint %q: %s (%s)", endpoint, err, validationContext.Source))
	}
}
