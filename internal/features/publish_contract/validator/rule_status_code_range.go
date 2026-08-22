package validator

import (
	"fmt"
)

const (
	MIN_STATUS_CODE = 100
	MAX_STATUS_CODE = 599
)

type statusCodeRangeRule struct{}

func (statusCodeRangeRule) Code() string { return "status.out_of_range" }

func (statusCodeRangeRule) Validate(value any, validationContext *ValidationContext) {
	statusCode, ok := value.(int)
	if !ok {
		return
	}

	if statusCode < MIN_STATUS_CODE || statusCode > MAX_STATUS_CODE {
		validationContext.AddViolation(fmt.Sprintf(
			"invalid status code %d at %s (%s)",
			statusCode,
			validationContext.Where,
			validationContext.Source,
		))
	}
}
