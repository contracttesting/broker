package validator

import (
	"fmt"

	"github.com/contracttesting/broker/internal/dsl"
)

type schemaTooDeepRule struct{}

func (schemaTooDeepRule) Code() string { return "schema.too_deep" }

func (schemaTooDeepRule) Validate(value any, validationContext *ValidationContext) {
	if _, ok := value.(dsl.Schema); !ok {
		return
	}

	if !validationContext.Depth.Exceeded() {
		return
	}

	validationContext.AddViolation(fmt.Sprintf(
		"schema %s is too deep with more than %d levels (%s)",
		validationContext.RootSchema,
		MAX_DEPTH,
		validationContext.Source,
	))
}
