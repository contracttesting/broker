package validator

import (
	"fmt"

	"github.com/contracttesting/broker/internal/features/publish_contract/dsl"
	"github.com/contracttesting/broker/internal/features/publish_contract/mapper/schemamapper"
)

type schemaTooDeepRule struct{}

func (schemaTooDeepRule) Code() string { return "schema.too_deep" }

func (schemaTooDeepRule) Validate(value any, contextualValidator *ContextualValidator) {
	if _, ok := value.(dsl.Schema); !ok {
		return
	}

	if !contextualValidator.depth.Exceeded() {
		return
	}

	contextualValidator.addViolation(fmt.Sprintf(
		"schema %s is too deep with more than %d levels (%s)",
		contextualValidator.rootSchema,
		schemamapper.MaxDepth,
		contextualValidator.source,
	))
}
