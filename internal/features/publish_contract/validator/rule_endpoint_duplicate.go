package validator

import (
	"fmt"

	"github.com/contracttesting/broker/internal/common"
)

type endpointDuplicateRule struct {
	seen map[string]bool
}

func (endpointDuplicateRule) Code() string { return "endpoint.duplicate" }

func (endpointDuplicateRule) Fresh() StatefulRule {
	return &endpointDuplicateRule{seen: map[string]bool{}}
}

func (r *endpointDuplicateRule) Validate(value any, validationContext *ValidationContext) {
	endpoint, ok := value.(string)
	if !ok {
		return
	}

	normalized := common.NormalizeEndpoint(endpoint)

	// duplicates only exist within one rest block of one file: the same endpoint in
	// another file or block is the resource duplicate rule's business
	scope := validationContext.Source + "|" + validationContext.Where + "|" + normalized

	if r.seen[scope] {
		validationContext.AddViolation(fmt.Sprintf("duplicate endpoint: %s declared twice in %s", normalized, validationContext.Source))

		return
	}

	r.seen[scope] = true
}
