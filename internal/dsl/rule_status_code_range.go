package dsl

import (
	"maps"
	"slices"
)

type statusCodeRangeRule struct{}

func (statusCodeRangeRule) Code() string { return "status.out-of-range" }

func (statusCodeRangeRule) CheckResponses(r Responses, vctx ValidationContext) {
	for _, statusCode := range slices.Sorted(maps.Keys(r)) {
		if statusCode < MIN_STATUS_CODE || statusCode > MAX_STATUS_CODE {
			vctx.Errs.Addf(
				"invalid status code %d at %s (%s)",
				statusCode,
				vctx.Pos.Where,
				vctx.Pos.Source,
			)
		}
	}
}
