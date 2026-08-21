package validator

import (
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/contracttesting/broker/internal/dsl"
	"github.com/contracttesting/broker/internal/model"
)

type DslValidator struct {
	ruleEntries []ruleEntry
}

type ruleEntry struct {
	segment string
	rule    Rule
}

func NewDslValidator() *DslValidator {
	return &DslValidator{ruleEntries: []ruleEntry{
		{segment: SegmentService, rule: serviceNameRule{}},
		{segment: SegmentEndpoint, rule: endpointSyntaxRule{}},
		{segment: SegmentRest, rule: endpointDuplicateRule{}},
		{segment: SegmentSchemas, rule: &schemaDuplicateRule{}},
		{segment: SegmentResource, rule: &resourceDuplicateRule{}},
		{segment: SegmentSchemaName, rule: schemaUnresolvedNameRule{}},
		{segment: SegmentSchema, rule: schemaUnresolvedRefRule{}},
		{segment: SegmentSchema, rule: schemaTooDeepRule{}},
		{segment: SegmentSchema, rule: schemaArrayWithoutItemsRule{}},
		{segment: SegmentSchema, rule: schemaInvalidTypeRule{}},
		{segment: SegmentResponses, rule: statusCodeRangeRule{}},
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
	violations := []string{}

	for _, fragment := range sortedBySource(fragments) {
		validationContext := NewValidationContext(fragment.Source, *contractIndex, rules)
		v.validateContract(fragment, validationContext)
		violations = append(violations, validationContext.Violations...)
	}

	return violations
}

func (v *DslValidator) validateContract(fragment dsl.Fragment, validationContext *ValidationContext) {
	root := dsl.NewResourcePath("")

	v.validateRest(fragment.Contract.Provides.Rest, "provides", root.Append("provides"), validationContext)

	for _, service := range slices.Sorted(maps.Keys(fragment.Contract.ConsumesServices)) {
		dispatch(validationContext, SegmentService, service)

		// what an invalid service name declares is unreachable anyway
		if model.ParticipantNameViolation(service) != "" {
			continue
		}

		v.validateRest(
			fragment.Contract.ConsumesServices[service].Rest,
			joinWhere("consumes", service),
			root.Append("consumes", service),
			validationContext,
		)
	}

	v.validateSchemas(fragment.Contract.Schemas, validationContext)
}

func (v *DslValidator) validateRest(rest dsl.Rest, where string, resourcePath dsl.ResourcePath, validationContext *ValidationContext) {
	validationContext.Where = where
	dispatch(validationContext, SegmentRest, rest)

	visited := map[string]bool{}

	for _, endpoint := range slices.Sorted(maps.Keys(rest)) {
		validationContext.Where = where
		dispatch(validationContext, SegmentEndpoint, endpoint)

		normalized := dsl.NormalizeEndpoint(endpoint)

		// what an invalid endpoint declares is unreachable anyway
		if endpointViolation(normalized) != "" {
			continue
		}

		// the second spelling of a same-file duplicate: the first one won, and
		// descending again would only pile resource noise on the same problem
		if visited[normalized] {
			continue
		}
		visited[normalized] = true

		v.validateMethod(rest[endpoint], where, normalized, resourcePath.Append("rest", normalized), validationContext)
	}
}

func (v *DslValidator) validateMethod(methods dsl.HttpMethods, where string, endpoint string, resourcePath dsl.ResourcePath, validationContext *ValidationContext) {
	// the pinned breadcrumb order is verb before endpoint: "provides GET /pets 200"
	v.validateGet(methods.Get, joinWhere(joinWhere(where, "GET"), endpoint), resourcePath.Append("get"), validationContext)
	v.validatePost(methods.Post, joinWhere(joinWhere(where, "POST"), endpoint), resourcePath.Append("post"), validationContext)
	v.validatePut(methods.Put, joinWhere(joinWhere(where, "PUT"), endpoint), resourcePath.Append("put"), validationContext)
	v.validateDelete(methods.Delete, joinWhere(joinWhere(where, "DELETE"), endpoint), resourcePath.Append("delete"), validationContext)
}

func (v *DslValidator) validateGet(getMethod dsl.GetMethod, where string, resourcePath dsl.ResourcePath, validationContext *ValidationContext) {
	v.validateResponses(getMethod.Responses, where, resourcePath.Append("responses"), validationContext)
}

func (v *DslValidator) validatePost(postMethod dsl.PostMethod, where string, resourcePath dsl.ResourcePath, validationContext *ValidationContext) {
	if postMethod.HasRequestBody() {
		v.validateRequestBody(postMethod.RequestBody, where, resourcePath, validationContext)
	}

	v.validateResponses(postMethod.Responses, where, resourcePath.Append("responses"), validationContext)
}

func (v *DslValidator) validatePut(putMethod dsl.PutMethod, where string, resourcePath dsl.ResourcePath, validationContext *ValidationContext) {
	if putMethod.HasRequestBody() {
		v.validateRequestBody(putMethod.RequestBody, where, resourcePath, validationContext)
	}

	v.validateResponses(putMethod.Responses, where, resourcePath.Append("responses"), validationContext)
}

func (v *DslValidator) validateDelete(deleteMethod dsl.DeleteMethod, where string, resourcePath dsl.ResourcePath, validationContext *ValidationContext) {
	v.validateResponses(deleteMethod.Responses, where, resourcePath.Append("responses"), validationContext)
}

func (v *DslValidator) validateRequestBody(schemaName string, where string, resourcePath dsl.ResourcePath, validationContext *ValidationContext) {
	requestPath := resourcePath.Append("request")

	validationContext.Where = joinWhere(where, "request")
	dispatch(validationContext, SegmentResource, requestPath.String())
	dispatch(validationContext, SegmentSchemaName, schemaName)
}

func (v *DslValidator) validateResponses(responses dsl.Responses, where string, resourcePath dsl.ResourcePath, validationContext *ValidationContext) {
	validationContext.Where = where
	dispatch(validationContext, SegmentResponses, responses)

	for _, statusCode := range slices.Sorted(maps.Keys(responses)) {
		status := strconv.Itoa(statusCode)
		statusPath := resourcePath.Append(status)

		validationContext.Where = joinWhere(where, status)
		dispatch(validationContext, SegmentResource, statusPath.String())
		dispatch(validationContext, SegmentSchemaName, responses[statusCode])
	}
}

func (v *DslValidator) validateSchemas(schemas dsl.SchemasMap, validationContext *ValidationContext) {
	validationContext.Where = ""
	dispatch(validationContext, SegmentSchemas, schemas)

	for _, name := range slices.Sorted(maps.Keys(schemas)) {
		validationContext.RootSchema = name
		v.validateSchema(schemas[name], name, DepthCounter{}, validationContext)
	}
}

func (v *DslValidator) validateSchema(schema dsl.Schema, path string, depth DepthCounter, validationContext *ValidationContext) {
	validationContext.Where = path
	validationContext.Depth = depth
	dispatch(validationContext, SegmentSchema, schema)

	// the descent stops where there is nothing sound to descend into; the message,
	// when one is due, came from the rules above
	if depth.Exceeded() {
		return
	}

	switch {
	case schema.IsRef():
		target, declared := validationContext.ContractIndex.Schema(schema.Ref)
		if !declared {
			return
		}

		v.validateSchema(target, path, depth.Deeper(), validationContext)

	case schema.IsArray():
		if schema.Items == nil {
			return
		}

		v.validateSchema(*schema.Items, path+"[]", depth.Deeper(), validationContext)

	case schema.IsObject():
		for _, name := range slices.Sorted(maps.Keys(schema.Properties)) {
			v.validateSchema(schema.Properties[name], path+"."+name, depth.Deeper(), validationContext)
		}
	}
}

func joinWhere(where, segment string) string {
	if where == "" || segment == "" {
		return where + segment
	}

	return where + " " + segment
}

func sortedBySource(fragments []dsl.Fragment) []dsl.Fragment {
	sorted := slices.Clone(fragments)

	slices.SortFunc(sorted, func(a, b dsl.Fragment) int {
		return strings.Compare(a.Source, b.Source)
	})

	return sorted
}
