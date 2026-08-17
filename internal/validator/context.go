package validator

import (
	"fmt"
	"slices"
	"strings"

	"github.com/contracttesting/broker/internal/dsl"
)

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
	Path   dsl.PropertyPath
	Root   string
	Depth  DepthCounter

	// an endpoint is visited before its methods but reads after them
	// ("provides GET /pets 200"), so it waits here for the method segment
	pendingEndpoint string

	// the resource key of the branch (direction, service when consumes, normalized
	// endpoint, method, request|status), grown by atResource as the walk descends
	resource dsl.ResourcePath
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
	v.Pos.Path = dsl.NewPropertyPath(name)
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

func (v ValidationContext) checkRest(r dsl.Rest) {
	for _, rule := range v.rules {
		if hook, ok := rule.(RestRule); ok {
			hook.CheckRest(r, v)
		}
	}
}

func (v ValidationContext) checkSchemas(s dsl.SchemasMap) {
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

func (v ValidationContext) checkSchema(s dsl.Schema) {
	for _, rule := range v.rules {
		if hook, ok := rule.(SchemaRule); ok {
			hook.CheckSchema(s, v)
		}
	}
}

func (v ValidationContext) checkResponses(r dsl.Responses) {
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

func sortedBySource(fragments []dsl.Fragment) []dsl.Fragment {
	sorted := slices.Clone(fragments)
	slices.SortFunc(sorted, func(a, b dsl.Fragment) int {
		return strings.Compare(a.Source, b.Source)
	})

	return sorted
}
