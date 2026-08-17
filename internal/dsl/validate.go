package dsl

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
)

// Validate walks every fragment and reports everything wrong with the contract at once:
// a publish is either rejected with the full list or handed to the build as valid DSL.
// It expects the fragments to have gone through Normalize.
func Validate(fragments []Fragment) []error {
	errs := &ErrorList{}
	index := buildIndex(fragments, errs)

	for _, fragment := range sortedBySource(fragments) {
		fragment.Contract.Validate(ValidationContext{
			Index: index,
			Errs:  errs,
			Pos:   Position{Source: fragment.Source},
		})
	}

	return errs.All()
}

// Index is what a rule needs to know about the contract as a whole: the schema
// namespace every fragment shares, and who declared what. It is built once, before the
// walk, and never changes during it.
type Index struct {
	schemas       SchemasMap
	schemaSources map[string]string
	resources     map[string]string
}

func (i *Index) Schema(name string) (Schema, bool) {
	schema, declared := i.schemas[name]

	return schema, declared
}

// buildIndex indexes the fragments in source order. Duplicates are a by-product of the
// indexing: the second declaration of a name or of a resource path is the one that
// cannot be inserted.
func buildIndex(fragments []Fragment, errs *ErrorList) *Index {
	index := &Index{
		schemas:       make(SchemasMap),
		schemaSources: make(map[string]string),
		resources:     make(map[string]string),
	}

	for _, fragment := range sortedBySource(fragments) {
		index.indexSchemas(fragment, errs)
	}

	for _, fragment := range sortedBySource(fragments) {
		index.indexResources(fragment, errs)
	}

	return index
}

func (i *Index) indexSchemas(fragment Fragment, errs *ErrorList) {
	for _, name := range slices.Sorted(maps.Keys(fragment.Contract.Schemas)) {
		if declaredIn, taken := i.schemaSources[name]; taken {
			errs.Addf("duplicate schema: %s declared in %s and %s", name, declaredIn, fragment.Source)

			continue
		}

		i.schemas[name] = fragment.Contract.Schemas[name]
		i.schemaSources[name] = fragment.Source
	}
}

func (i *Index) indexResources(fragment Fragment, errs *ErrorList) {
	root := NewResourcePath("")

	providesPath := root.Append("provides")
	i.indexRest(fragment.Contract.Provides.Rest, providesPath, fragment.Source, errs)

	for _, service := range slices.Sorted(maps.Keys(fragment.Contract.ConsumesServices)) {
		consumesPath := root.Append("consumes", service)
		i.indexRest(fragment.Contract.ConsumesServices[service].Rest, consumesPath, fragment.Source, errs)
	}
}

func (i *Index) indexRest(rest Rest, base ResourcePath, source string, errs *ErrorList) {
	for _, endpoint := range slices.Sorted(maps.Keys(rest)) {
		// an endpoint the walk will reject declares no resource, and its path would not
		// parse back into one
		if validateEndpoint(endpoint) != nil {
			continue
		}

		methods := rest[endpoint]
		endpointPath := base.Append("rest", endpoint)

		i.indexResponses(endpointPath.Append("get", "responses"), methods.Get.Responses, source, errs)

		if methods.Post.HasRequestBody() {
			i.indexResource(endpointPath.Append("post", "request"), source, errs)
		}
		i.indexResponses(endpointPath.Append("post", "responses"), methods.Post.Responses, source, errs)

		if methods.Put.HasRequestBody() {
			i.indexResource(endpointPath.Append("put", "request"), source, errs)
		}
		i.indexResponses(endpointPath.Append("put", "responses"), methods.Put.Responses, source, errs)

		i.indexResponses(endpointPath.Append("delete", "responses"), methods.Delete.Responses, source, errs)
	}
}

func (i *Index) indexResponses(base ResourcePath, responses Responses, source string, errs *ErrorList) {
	for _, statusCode := range slices.Sorted(maps.Keys(responses)) {
		i.indexResource(base.Append(strconv.Itoa(statusCode)), source, errs)
	}
}

func (i *Index) indexResource(resourcePath ResourcePath, source string, errs *ErrorList) {
	if declaredIn, taken := i.resources[resourcePath.String()]; taken {
		errs.Addf(
			"duplicate resource: %s declared in %s and %s",
			resourcePath.ToResource(nil).Describe(),
			declaredIn,
			source,
		)

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
}

type ValidationContext struct {
	Index *Index
	Errs  *ErrorList
	Pos   Position
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

func joinWhere(where, segment string) string {
	if where == "" || segment == "" {
		return where + segment
	}

	return where + " " + segment
}

// validateSchemaName checks that a request or response names a schema of the namespace.
// The schema itself is walked once, from its declaration, so nothing descends here.
func validateSchemaName(vctx ValidationContext, name string) {
	if _, declared := vctx.Index.Schema(name); !declared {
		vctx.Errs.Addf(
			"unresolved schema name: %s referenced at %s (%s)",
			name,
			vctx.Pos.Where,
			vctx.Pos.Source,
		)
	}
}

func sortedBySource(fragments []Fragment) []Fragment {
	sorted := slices.Clone(fragments)
	slices.SortFunc(sorted, func(a, b Fragment) int {
		return strings.Compare(a.Source, b.Source)
	})

	return sorted
}
