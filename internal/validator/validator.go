package validator

import (
	"fmt"

	"github.com/contracttesting/broker/internal/dsl"
)

type DslValidator struct {
	ruleEntries []ruleEntry
}

type ruleEntry struct {
	segment   string
	rule      Rule
	invariant bool
}

func NewDslValidator() *DslValidator {
	return &DslValidator{ruleEntries: []ruleEntry{
		{segment: SegmentEndpoint, rule: endpointSyntaxRule{}, invariant: true},
		{segment: SegmentRest, rule: endpointDuplicateRule{}, invariant: true},
		{segment: SegmentSchemas, rule: &schemaDuplicateRule{}, invariant: true},
		{segment: SegmentResource, rule: &resourceDuplicateRule{}, invariant: true},
		{segment: SegmentSchemaName, rule: schemaUnresolvedNameRule{}, invariant: true},
		{segment: SegmentSchema, rule: schemaUnresolvedRefRule{}, invariant: true},
		{segment: SegmentSchema, rule: schemaTooDeepRule{}, invariant: true},
		{segment: SegmentSchema, rule: schemaArrayWithoutItemsRule{}, invariant: true},
		{segment: SegmentSchema, rule: schemaInvalidTypeRule{}, invariant: true},
		{segment: SegmentResponses, rule: statusCodeRangeRule{}, invariant: false},
	}}
}

// freshRules builds the execution's rule set, indexed by the segment each entry is
// registered at. Stateful rules enter through Fresh(): one instance per execution,
// never shared across concurrent publishes.
func (v *DslValidator) freshRules() map[string][]Rule {
	rules := make(map[string][]Rule, len(v.ruleEntries))

	for _, entry := range v.ruleEntries {
		rule := entry.rule

		if stateful, ok := rule.(StatefulRule); ok {
			rule = stateful.Fresh()
		}

		rules[entry.segment] = append(rules[entry.segment], rule)
	}

	return rules
}

func (v *DslValidator) Validate(fragments []dsl.Fragment) []string {
	contractIndex := NewContractIndex(fragments)
	rules := v.freshRules()

	for _, fragment := range sortedBySource(fragments) {
		validationContext := NewValidationContext(fragment.Source, *contractIndex, rules)
		v.validateContract(fragment, validationContext)
	}

	return []string{}
}

func (v *DslValidator) validateContract(fragment dsl.Fragment, validationContext *ValidationContext) {
	v.validateProvides(fragment.Contract.Provides, validationContext)

	for _, consumes := range fragment.Contract.ConsumesServices {
		v.validateConsumes(consumes, validationContext)
	}
}

func (v *DslValidator) validateProvides(provides dsl.Provides, validationContext *ValidationContext) {
	for endpoint, methods := range provides.Rest {
		v.validateEndpoint(endpoint, validationContext)
		v.validateMethod(methods, validationContext)
	}
}

func (v *DslValidator) validateEndpoint(endpoint string, validationContext *ValidationContext) {
}

func (v *DslValidator) validateConsumes(consumes dsl.Consumes, validationContext *ValidationContext) {
}

func (v *DslValidator) validateMethod(methods dsl.HttpMethods, validationContext *ValidationContext) {
	v.validateGet(methods.Get, validationContext)
	v.validatePost(methods.Post, validationContext)
	v.validatePut(methods.Put, validationContext)
	v.validateDelete(methods.Delete, validationContext)

}

func (v *DslValidator) validateGet(getMethod dsl.GetMethod, validationContext *ValidationContext) {
	for statusCode, schemaName := range getMethod.Responses {
		fmt.Println(statusCode, schemaName)
	}
}

func (v *DslValidator) validatePost(postMethod dsl.PostMethod, validationContext *ValidationContext) {
	if postMethod.HasRequestBody() {
		fmt.Println("request", postMethod.RequestBody)
	}

	for statusCode, schemaName := range postMethod.Responses {
		fmt.Println("response", statusCode, schemaName)
	}
}

func (v *DslValidator) validatePut(putMethod dsl.PutMethod, validationContext *ValidationContext) {
	if putMethod.HasRequestBody() {
		fmt.Println("request", putMethod.RequestBody)
	}

	for statusCode, schemaName := range putMethod.Responses {
		fmt.Println(statusCode, schemaName)
	}
}

func (v *DslValidator) validateDelete(deleteMethod dsl.DeleteMethod, validationContext *ValidationContext) {
	for statusCode, schemaName := range deleteMethod.Responses {
		fmt.Println(statusCode, schemaName)
	}
}

func (v *DslValidator) validateSchema(schema dsl.Schema, validationContext *ValidationContext) {}
