package validator

import (
	"fmt"
	"maps"
	"slices"

	"github.com/contracttesting/broker/internal/dsl"
)

const (
	MIN_STATUS_CODE = 100
	MAX_STATUS_CODE = 599
)

type statusCodeRangeRule struct{}

func (statusCodeRangeRule) Code() string { return "status.out_of_range" }

func (statusCodeRangeRule) Validate(segment any, validationContext *ValidationContext) {
	responses, ok := segment.(dsl.Responses)
	if !ok {
		return
	}

	for _, statusCode := range slices.Sorted(maps.Keys(responses)) {
		if statusCode < MIN_STATUS_CODE || statusCode > MAX_STATUS_CODE {
			validationContext.AddViolation(fmt.Sprintf(
				"invalid status code %d at %s (%s)",
				statusCode,
				validationContext.Segment,
				validationContext.Source,
			))
		}
	}
}
