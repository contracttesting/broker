package dsl

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
)

// Index is what a rule needs to know about the contract as a whole: the schema
// namespace every fragment shares, and the resources they declare. It is built once,
// before the walk, and never changes during it. It exists for O(1) lookup: it merges
// silently, first entry winning — duplicate detection is the rules' job.
type Index struct {
	schemas   SchemasMap
	resources map[string]string
}

func (i *Index) Schema(name string) (Schema, bool) {
	schema, declared := i.schemas[name]

	return schema, declared
}

func buildIndex(fragments []Fragment) *Index {
	index := &Index{
		schemas:   make(SchemasMap),
		resources: make(map[string]string),
	}

	for _, fragment := range sortedBySource(fragments) {
		index.indexSchemas(fragment)
	}

	for _, fragment := range sortedBySource(fragments) {
		index.indexResources(fragment)
	}

	return index
}

func (i *Index) indexSchemas(fragment Fragment) {
	for _, name := range slices.Sorted(maps.Keys(fragment.Contract.Schemas)) {
		if _, taken := i.schemas[name]; taken {
			continue
		}

		i.schemas[name] = fragment.Contract.Schemas[name]
	}
}

func (i *Index) indexResources(fragment Fragment) {
	root := NewResourcePath("")

	providesPath := root.Append("provides")
	i.indexRest(fragment.Contract.Provides.Rest, providesPath, fragment.Source)

	for _, service := range slices.Sorted(maps.Keys(fragment.Contract.ConsumesServices)) {
		consumesPath := root.Append("consumes", service)
		i.indexRest(fragment.Contract.ConsumesServices[service].Rest, consumesPath, fragment.Source)
	}
}

func (i *Index) indexRest(rest Rest, base ResourcePath, source string) {
	for _, endpoint := range slices.Sorted(maps.Keys(rest)) {
		// an endpoint the walk will reject declares no resource, and its path would not
		// parse back into one
		normalized := normalizeEndpoint(endpoint)
		if endpointViolation(normalized) != "" {
			continue
		}

		methods := rest[endpoint]
		endpointPath := base.Append("rest", normalized)

		i.indexResponses(endpointPath.Append("get", "responses"), methods.Get.Responses, source)

		if methods.Post.HasRequestBody() {
			i.indexResource(endpointPath.Append("post", "request"), source)
		}
		i.indexResponses(endpointPath.Append("post", "responses"), methods.Post.Responses, source)

		if methods.Put.HasRequestBody() {
			i.indexResource(endpointPath.Append("put", "request"), source)
		}
		i.indexResponses(endpointPath.Append("put", "responses"), methods.Put.Responses, source)

		i.indexResponses(endpointPath.Append("delete", "responses"), methods.Delete.Responses, source)
	}
}

func (i *Index) indexResponses(base ResourcePath, responses Responses, source string) {
	for _, statusCode := range slices.Sorted(maps.Keys(responses)) {
		i.indexResource(base.Append(strconv.Itoa(statusCode)), source)
	}
}

func (i *Index) indexResource(resourcePath ResourcePath, source string) {
	if _, taken := i.resources[resourcePath.String()]; taken {
		return
	}

	i.resources[resourcePath.String()] = source
}

// ErrorList collects the violations of a whole publish: it is the only state the walk
// shares and mutates.
type ErrorList struct {
	errors []error
}

func (e *ErrorList) Addf(format string, args ...any) {
	e.errors = append(e.errors, fmt.Errorf(format, args...))
}

func (e *ErrorList) All() []error {
	return e.errors
}

// Position is where the walk stands. It travels by value: every descent hands its
// callee a copy, so a branch never leaks its position to the branches beside it.
type Position struct {
	Source string
	Where  string
	Path   PropertyPath
	Root   string
	Depth  DepthCounter

	// an endpoint is visited before its methods but reads after them
	// ("provides GET /pets 200"), so it waits here for the method segment
	pendingEndpoint string

	// the resource key of the branch (direction, service when consumes, normalized
	// endpoint, method, request|status), grown by atResource as the walk descends
	resource ResourcePath
}

type ValidationContext struct {
	Index *Index
	Errs  *ErrorList
	Pos   Position

	// the rule set of this execution: stateless rules shared, stateful ones fresh
	rules []Rule
}

// At advances the breadcrumb that names a resource in an error message.
func (v ValidationContext) At(segment string) ValidationContext {
	v.Pos.Where = joinWhere(v.Pos.Where, segment)

	if v.Pos.pendingEndpoint != "" {
		v.Pos.Where = joinWhere(v.Pos.Where, v.Pos.pendingEndpoint)
		v.Pos.pendingEndpoint = ""
	}

	return v
}

func (v ValidationContext) AtEndpoint(endpoint string) ValidationContext {
	v.Pos.pendingEndpoint = endpoint

	return v
}

// AtSchema starts a fresh descent at a declared schema: it is the root of the depth
// budget and of the property path that ref errors quote.
func (v ValidationContext) AtSchema(name string) ValidationContext {
	v.Pos.Root = name
	v.Pos.Path = NewPropertyPath(name)
	v.Pos.Depth = DepthCounter{}

	return v
}

func (v ValidationContext) AtProperty(name string) ValidationContext {
	v.Pos.Path = v.Pos.Path.Append(name)

	return v
}

func (v ValidationContext) AtItems() ValidationContext {
	v.Pos.Path = v.Pos.Path.AppendArray()

	return v
}

func (v ValidationContext) Deeper() ValidationContext {
	v.Pos.Depth = v.Pos.Depth.Deeper()

	return v
}

// atResource grows the resource key of the branch, mirroring the keys buildIndex
// builds.
func (v ValidationContext) atResource(parts ...string) ValidationContext {
	v.Pos.resource = v.Pos.resource.Append(parts...)

	return v
}

func (v ValidationContext) checkEndpoint(endpoint string) {
	for _, rule := range v.rules {
		if hook, ok := rule.(EndpointRule); ok {
			hook.CheckEndpoint(endpoint, v)
		}
	}
}

func (v ValidationContext) checkRest(r Rest) {
	for _, rule := range v.rules {
		if hook, ok := rule.(RestRule); ok {
			hook.CheckRest(r, v)
		}
	}
}

func (v ValidationContext) checkSchemas(s SchemasMap) {
	for _, rule := range v.rules {
		if hook, ok := rule.(SchemasRule); ok {
			hook.CheckSchemas(s, v)
		}
	}
}

func (v ValidationContext) checkResource() {
	for _, rule := range v.rules {
		if hook, ok := rule.(ResourceRule); ok {
			hook.CheckResource(v.Pos.resource.String(), v)
		}
	}
}

func (v ValidationContext) checkSchemaName(name string) {
	for _, rule := range v.rules {
		if hook, ok := rule.(SchemaNameRule); ok {
			hook.CheckSchemaName(name, v)
		}
	}
}

func (v ValidationContext) checkSchema(s Schema) {
	for _, rule := range v.rules {
		if hook, ok := rule.(SchemaRule); ok {
			hook.CheckSchema(s, v)
		}
	}
}

func (v ValidationContext) checkResponses(r Responses) {
	for _, rule := range v.rules {
		if hook, ok := rule.(ResponsesRule); ok {
			hook.CheckResponses(r, v)
		}
	}
}

func joinWhere(where, segment string) string {
	if where == "" || segment == "" {
		return where + segment
	}

	return where + " " + segment
}

func sortedBySource(fragments []Fragment) []Fragment {
	sorted := slices.Clone(fragments)
	slices.SortFunc(sorted, func(a, b Fragment) int {
		return strings.Compare(a.Source, b.Source)
	})

	return sorted
}
