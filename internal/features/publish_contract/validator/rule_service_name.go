package validator

import (
	"fmt"

	"github.com/contracttesting/broker/internal/model"
)

// A consumed service names the participant it expects on the other side, so the name
// is judged by the participant spelling rule.
type serviceNameRule struct{}

func (serviceNameRule) Code() string { return "service.name_syntax" }

func (serviceNameRule) Validate(segment any, validationContext *ValidationContext) {
	service, ok := segment.(string)
	if !ok {
		return
	}

	if reason := model.ParticipantNameViolation(service); reason != "" {
		validationContext.AddViolation(fmt.Sprintf("invalid service name %q: %s (%s)", service, reason, validationContext.Source))
	}
}
