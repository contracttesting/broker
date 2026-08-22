package validator

import (
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/contracttesting/broker/internal/common"
	"github.com/contracttesting/broker/internal/dsl"
	"github.com/contracttesting/broker/internal/validations"
)

type DslValidator struct {
	rules map[string][]Rule
}

func NewDslValidator() *DslValidator {
	return &DslValidator{rules: map[string][]Rule{
		SegmentServiceName:       {serviceNameRule{}},
		SegmentEndpoint:          {endpointSyntaxRule{}, &endpointDuplicateRule{}},
		SegmentResource:          {&resourceDuplicateRule{}},
		SegmentSchemaName:        {schemaUnresolvedNameRule{}},
		SegmentSchemaDeclaration: {&schemaDuplicateRule{}},
		SegmentSchema:            {schemaUnresolvedRefRule{}, schemaTooDeepRule{}, schemaArrayWithoutItemsRule{}, schemaInvalidTypeRule{}},
		SegmentStatusCode:        {statusCodeRangeRule{}},
	}}
}

func (v *DslValidator) freshRules() map[string][]Rule {
	rules := make(map[string][]Rule, len(v.rules))

	for segment, segmentRules := range v.rules {
		fresh := make([]Rule, len(segmentRules))

		for position, rule := range segmentRules {
			if stateful, ok := rule.(StatefulRule); ok {
				rule = stateful.Fresh()
			}

			fresh[position] = rule
		}

		rules[segment] = fresh
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

	for _, serviceName := range slices.Sorted(maps.Keys(fragment.Contract.ConsumesServices)) {
		validationContext.Validate(SegmentServiceName, serviceName)

		if validations.ParticipantName(serviceName) != nil {
			continue
		}

		v.validateRest(
			fragment.Contract.ConsumesServices[serviceName].Rest,
			joinWhere("consumes", serviceName),
			root.Append("consumes", serviceName),
			validationContext,
		)
	}

	v.validateSchemas(fragment.Contract.Schemas, validationContext)
}

func (v *DslValidator) validateRest(rest dsl.Rest, where string, resourcePath dsl.ResourcePath, validationContext *ValidationContext) {
	visited := map[string]bool{}

	for _, endpoint := range slices.Sorted(maps.Keys(rest)) {
		validationContext.Where = where
		validationContext.Validate(SegmentEndpoint, endpoint)

		normalized := common.NormalizeEndpoint(endpoint)

		// what an invalid endpoint declares is unreachable anyway
		if validations.Endpoint(normalized) != nil {
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
	validationContext.Validate(SegmentResource, requestPath.String())
	validationContext.Validate(SegmentSchemaName, schemaName)
}

func (v *DslValidator) validateResponses(responses dsl.Responses, where string, resourcePath dsl.ResourcePath, validationContext *ValidationContext) {
	for _, statusCode := range slices.Sorted(maps.Keys(responses)) {
		status := strconv.Itoa(statusCode)
		statusPath := resourcePath.Append(status)

		validationContext.Where = where
		validationContext.Validate(SegmentStatusCode, statusCode)

		validationContext.Where = joinWhere(where, status)
		validationContext.Validate(SegmentResource, statusPath.String())
		validationContext.Validate(SegmentSchemaName, responses[statusCode])
	}
}

func (v *DslValidator) validateSchemas(schemas dsl.SchemasMap, validationContext *ValidationContext) {
	for _, name := range slices.Sorted(maps.Keys(schemas)) {
		validationContext.Where = ""
		validationContext.Validate(SegmentSchemaDeclaration, name)

		validationContext.RootSchema = name
		v.validateSchema(schemas[name], name, DepthCounter{}, validationContext)
	}
}

func (v *DslValidator) validateSchema(schema dsl.Schema, path string, depth DepthCounter, validationContext *ValidationContext) {
	validationContext.Where = path
	validationContext.Depth = depth
	validationContext.Validate(SegmentSchema, schema)

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
