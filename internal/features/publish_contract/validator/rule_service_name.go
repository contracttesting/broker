package validator

import (
	"fmt"

	"github.com/contracttesting/broker/internal/validations"
)

// A consumed service names the participant it expects on the other side, so the name
// is judged by the participant spelling rule.
type serviceNameRule struct{}

func (serviceNameRule) Code() string { return "service.name_syntax" }

func (serviceNameRule) Validate(value any, validationContext *ValidationContext) {
	serviceName, ok := value.(string)
	if !ok {
		return
	}

	if err := validations.ParticipantName(serviceName); err != nil {
		validationContext.AddViolation(fmt.Sprintf("invalid service name %q: %s (%s)", serviceName, err, validationContext.Source))
	}
}
