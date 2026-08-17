package validator

import (
	"maps"
	"slices"
	"strconv"

	"github.com/contracttesting/broker/internal/dsl"
)

// Index is what a rule needs to know about the contract as a whole: the schema
// namespace every fragment shares, and the resources they declare. It is built once,
// before the walk, and never changes during it. It exists for O(1) lookup: it merges
// silently, first entry winning — duplicate detection is the rules' job.
type Index struct {
	schemas   dsl.SchemasMap
	resources map[string]string
}

func (i *Index) Schema(name string) (dsl.Schema, bool) {
	schema, declared := i.schemas[name]

	return schema, declared
}

func buildIndex(fragments []dsl.Fragment) *Index {
	index := &Index{
		schemas:   make(dsl.SchemasMap),
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

func (i *Index) indexSchemas(fragment dsl.Fragment) {
	for _, name := range slices.Sorted(maps.Keys(fragment.Contract.Schemas)) {
		if _, taken := i.schemas[name]; taken {
			continue
		}

		i.schemas[name] = fragment.Contract.Schemas[name]
	}
}

func (i *Index) indexResources(fragment dsl.Fragment) {
	root := dsl.NewResourcePath("")

	providesPath := root.Append("provides")
	i.indexRest(fragment.Contract.Provides.Rest, providesPath, fragment.Source)

	for _, service := range slices.Sorted(maps.Keys(fragment.Contract.ConsumesServices)) {
		consumesPath := root.Append("consumes", service)
		i.indexRest(fragment.Contract.ConsumesServices[service].Rest, consumesPath, fragment.Source)
	}
}

func (i *Index) indexRest(rest dsl.Rest, base dsl.ResourcePath, source string) {
	for _, endpoint := range slices.Sorted(maps.Keys(rest)) {
		// an endpoint the walk will reject declares no resource, and its path would not
		// parse back into one
		normalized := dsl.NormalizeEndpoint(endpoint)
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

func (i *Index) indexResponses(base dsl.ResourcePath, responses dsl.Responses, source string) {
	for _, statusCode := range slices.Sorted(maps.Keys(responses)) {
		i.indexResource(base.Append(strconv.Itoa(statusCode)), source)
	}
}

func (i *Index) indexResource(resourcePath dsl.ResourcePath, source string) {
	if _, taken := i.resources[resourcePath.String()]; taken {
		return
	}

	i.resources[resourcePath.String()] = source
}
