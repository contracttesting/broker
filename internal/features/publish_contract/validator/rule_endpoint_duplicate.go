package validator

import (
	"fmt"

	"github.com/contracttesting/broker/internal/common"
)

type endpointDuplicateRule struct {
	seen map[string]bool
}

func (endpointDuplicateRule) Code() string { return "endpoint.duplicate" }

func (r *endpointDuplicateRule) Validate(value any, contextualValidator *ContextualValidator) {
	endpoint, ok := value.(string)
	if !ok {
		return
	}

	normalized := common.NormalizeEndpoint(endpoint)

	// duplicates only exist within one rest block of one file: the same endpoint in
	// another file or block is the resource duplicate rule's business
	scope := contextualValidator.source + "|" + contextualValidator.where + "|" + normalized

	if r.seen[scope] {
		contextualValidator.addViolation(fmt.Sprintf("duplicate endpoint: %s declared twice in %s", normalized, contextualValidator.source))

		return
	}

	r.seen[scope] = true
}
