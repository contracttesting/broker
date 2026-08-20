package validator

import (
	"maps"
	"slices"
	"strconv"

	"github.com/contracttesting/broker/internal/dsl"
)

type ContractIndex struct {
	schemas   dsl.SchemasMap
	resources map[string]string
}

func NewContractIndex(fragments []dsl.Fragment) *ContractIndex {
	index := &ContractIndex{
		schemas:   make(dsl.SchemasMap),
		resources: make(map[string]string),
	}

	for _, fragment := range sortedBySource(fragments) {
		index.indexSchemas(fragment)
	}

	for _, fragment := range sortedBySource(fragments) {
		index.IndexResources(fragment)
	}

	return index
}

func (i *ContractIndex) Schema(name string) (dsl.Schema, bool) {
	schema, declared := i.schemas[name]

	return schema, declared
}

func (i *ContractIndex) indexSchemas(fragment dsl.Fragment) {
	for _, name := range slices.Sorted(maps.Keys(fragment.Contract.Schemas)) {
		if _, taken := i.schemas[name]; taken {
			continue
		}

		i.schemas[name] = fragment.Contract.Schemas[name]
	}
}

func (i *ContractIndex) IndexResources(fragment dsl.Fragment) {
	root := dsl.NewResourcePath("")

	providesPath := root.Append("provides")
	i.indexRest(fragment.Contract.Provides.Rest, providesPath, fragment.Source)

	for _, service := range slices.Sorted(maps.Keys(fragment.Contract.ConsumesServices)) {
		consumesPath := root.Append("consumes", service)
		i.indexRest(fragment.Contract.ConsumesServices[service].Rest, consumesPath, fragment.Source)
	}
}

func (i *ContractIndex) indexRest(rest dsl.Rest, base dsl.ResourcePath, source string) {
	for _, endpoint := range slices.Sorted(maps.Keys(rest)) {
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

func (i *ContractIndex) indexResponses(base dsl.ResourcePath, responses dsl.Responses, source string) {
	for _, statusCode := range slices.Sorted(maps.Keys(responses)) {
		i.indexResource(base.Append(strconv.Itoa(statusCode)), source)
	}
}

func (i *ContractIndex) indexResource(resourcePath dsl.ResourcePath, source string) {
	if _, taken := i.resources[resourcePath.String()]; taken {
		return
	}

	i.resources[resourcePath.String()] = source
}
