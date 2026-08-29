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

type ContextualValidator struct {
	rules map[string][]Rule

	source        string
	where         string
	rootSchema    string
	depth         DepthCounter
	contractIndex ContractIndex
	violations    []string
}

func NewContextualValidator() *ContextualValidator {
	return &ContextualValidator{
		rules: map[string][]Rule{
			SegmentServiceName:       {serviceNameRule{}},
			SegmentEndpoint:          {endpointSyntaxRule{}},
			SegmentResource:          {&resourceDuplicateRule{seen: map[string]string{}}},
			SegmentResourceSchema:    {&resourceTypeConflictRule{seen: map[string]map[string]typedProperty{}}},
			SegmentSchemaName:        {schemaUnresolvedNameRule{}},
			SegmentSchemaDeclaration: {&schemaDuplicateRule{seen: map[string]string{}}},
			SegmentSchema:            {schemaUnresolvedRefRule{}, schemaTooDeepRule{}, schemaArrayWithoutItemsRule{}, schemaInvalidTypeRule{}},
			SegmentStatusCode:        {statusCodeRangeRule{}},
		},
		violations: []string{},
	}
}

func (v *ContextualValidator) Validate(fragments []dsl.Fragment) []string {
	v.contractIndex = *NewContractIndex(fragments)

	for _, fragment := range sortedBySource(fragments) {
		v.source = fragment.Source
		v.validateContract(fragment)
	}

	return v.violations
}

func (v *ContextualValidator) validateBySegment(segment string, value any) {
	for _, rule := range v.rules[segment] {
		rule.Validate(value, v)
	}
}

func (v *ContextualValidator) addViolation(violation string) {
	v.violations = append(v.violations, violation)
}

func (v *ContextualValidator) validateContract(fragment dsl.Fragment) {
	root := dsl.NewResourcePath("")

	v.validateRest(fragment.Contract.Provides.Rest, "provides", root.Append("provides"))

	for _, serviceName := range slices.Sorted(maps.Keys(fragment.Contract.ConsumesServices)) {
		v.validateBySegment(SegmentServiceName, serviceName)

		if validations.ParticipantName(serviceName) != nil {
			continue
		}

		v.validateRest(
			fragment.Contract.ConsumesServices[serviceName].Rest,
			joinWhere("consumes", serviceName),
			root.Append("consumes", serviceName),
		)
	}

	v.validateSchemas(fragment.Contract.Schemas)
}

func (v *ContextualValidator) validateRest(rest dsl.Rest, where string, resourcePath dsl.ResourcePath) {
	for _, endpoint := range slices.Sorted(maps.Keys(rest)) {
		v.where = where
		v.validateBySegment(SegmentEndpoint, endpoint)

		normalized := common.NormalizeEndpoint(endpoint)

		if validations.Endpoint(normalized) != nil {
			continue
		}

		v.validateMethod(rest[endpoint], where, normalized, resourcePath.Append("rest", normalized))
	}
}

func (v *ContextualValidator) validateMethod(methods dsl.HttpMethods, where string, endpoint string, resourcePath dsl.ResourcePath) {
	v.validateGet(methods.Get, joinWhere(joinWhere(where, "GET"), endpoint), resourcePath.Append("get"))
	v.validatePost(methods.Post, joinWhere(joinWhere(where, "POST"), endpoint), resourcePath.Append("post"))
	v.validatePut(methods.Put, joinWhere(joinWhere(where, "PUT"), endpoint), resourcePath.Append("put"))
	v.validateDelete(methods.Delete, joinWhere(joinWhere(where, "DELETE"), endpoint), resourcePath.Append("delete"))
}

func (v *ContextualValidator) validateGet(getMethod dsl.GetMethod, where string, resourcePath dsl.ResourcePath) {
	v.validateResponses(getMethod.Responses, where, resourcePath.Append("responses"))
}

func (v *ContextualValidator) validatePost(postMethod dsl.PostMethod, where string, resourcePath dsl.ResourcePath) {
	if postMethod.HasRequestBody() {
		v.validateRequestBody(postMethod.RequestBody, where, resourcePath)
	}

	v.validateResponses(postMethod.Responses, where, resourcePath.Append("responses"))
}

func (v *ContextualValidator) validatePut(putMethod dsl.PutMethod, where string, resourcePath dsl.ResourcePath) {
	if putMethod.HasRequestBody() {
		v.validateRequestBody(putMethod.RequestBody, where, resourcePath)
	}

	v.validateResponses(putMethod.Responses, where, resourcePath.Append("responses"))
}

func (v *ContextualValidator) validateDelete(deleteMethod dsl.DeleteMethod, where string, resourcePath dsl.ResourcePath) {
	v.validateResponses(deleteMethod.Responses, where, resourcePath.Append("responses"))
}

func (v *ContextualValidator) validateRequestBody(schemaName string, where string, resourcePath dsl.ResourcePath) {
	requestPath := resourcePath.Append("request")

	v.where = joinWhere(where, "request")
	v.validateBySegment(SegmentResource, requestPath.String())
	v.validateBySegment(SegmentSchemaName, schemaName)
	v.validateBySegment(SegmentResourceSchema, ResourceSchema{requestPath.String(), schemaName})
}

func (v *ContextualValidator) validateResponses(responses dsl.Responses, where string, resourcePath dsl.ResourcePath) {
	for _, statusCode := range slices.Sorted(maps.Keys(responses)) {
		status := strconv.Itoa(statusCode)
		statusPath := resourcePath.Append(status)

		v.where = where
		v.validateBySegment(SegmentStatusCode, statusCode)

		v.where = joinWhere(where, status)
		v.validateBySegment(SegmentResource, statusPath.String())
		v.validateBySegment(SegmentSchemaName, responses[statusCode])
		v.validateBySegment(SegmentResourceSchema, ResourceSchema{statusPath.String(), responses[statusCode]})
	}
}

func (v *ContextualValidator) validateSchemas(schemas dsl.SchemasMap) {
	for _, name := range slices.Sorted(maps.Keys(schemas)) {
		v.where = ""
		v.validateBySegment(SegmentSchemaDeclaration, name)

		v.rootSchema = name
		v.validateSchema(schemas[name], name, DepthCounter{})
	}
}

func (v *ContextualValidator) validateSchema(schema dsl.Schema, path string, depth DepthCounter) {
	v.where = path
	v.depth = depth
	v.validateBySegment(SegmentSchema, schema)

	if depth.Exceeded() {
		return
	}

	switch {
	case schema.IsRef():
		target, declared := v.contractIndex.Schema(schema.Ref)
		if !declared {
			return
		}

		v.validateSchema(target, path, depth.Deeper())

	case schema.IsArray():
		if schema.Items == nil {
			return
		}

		v.validateSchema(*schema.Items, path+"[]", depth.Deeper())

	case schema.IsObject():
		for _, name := range slices.Sorted(maps.Keys(schema.Properties)) {
			v.validateSchema(schema.Properties[name], path+"."+name, depth.Deeper())
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
