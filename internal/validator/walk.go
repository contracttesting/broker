package validator

import (
	"maps"
	"slices"
	"strconv"

	"github.com/contracttesting/broker/internal/dsl"
)

// The walk is external traversal over the dsl structs: the DSL stays pure data, the
// validator owns where to go and which hooks to dispatch at each node. The messages
// come from the rules; the gates that stop a dead-end descent are the walker's own.
func walkContract(c dsl.Contract, vctx ValidationContext) {
	walkProvides(c.Provides, vctx.At("provides").atResource("provides"))
	walkConsumesServices(c.ConsumesServices, vctx)
	walkSchemas(c.Schemas, vctx)
}

// Provides.Message and Consumes.Message are placeholders for messaging contracts:
// nothing downstream reads them, so the walk stops at Rest.
func walkProvides(p dsl.Provides, vctx ValidationContext) {
	walkRest(p.Rest, vctx)
}

func walkConsumesServices(c dsl.ConsumesServicesMap, vctx ValidationContext) {
	for _, service := range slices.Sorted(maps.Keys(c)) {
		walkConsumes(c[service], vctx.At("consumes").At(service).atResource("consumes", service))
	}
}

func walkConsumes(c dsl.Consumes, vctx ValidationContext) {
	walkRest(c.Rest, vctx)
}

func walkRest(r dsl.Rest, vctx ValidationContext) {
	vctx.checkRest(r)

	visited := make(map[string]bool)

	for _, endpoint := range slices.Sorted(maps.Keys(r)) {
		vctx.checkEndpoint(endpoint)

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

		walkMethods(r[endpoint], vctx.AtEndpoint(normalized).atResource("rest", normalized))
	}
}

func walkMethods(m dsl.HttpMethods, vctx ValidationContext) {
	walkGet(m.Get, vctx.At("GET").atResource("get"))
	walkPost(m.Post, vctx.At("POST").atResource("post"))
	walkPut(m.Put, vctx.At("PUT").atResource("put"))
	walkDelete(m.Delete, vctx.At("DELETE").atResource("delete"))
}

func walkGet(g dsl.GetMethod, vctx ValidationContext) {
	walkResponses(g.Responses, vctx)
}

func walkPost(p dsl.PostMethod, vctx ValidationContext) {
	if p.HasRequestBody() {
		rctx := vctx.At("request").atResource("request")
		rctx.checkResource()
		rctx.checkSchemaName(p.RequestBody)
	}

	walkResponses(p.Responses, vctx)
}

func walkPut(p dsl.PutMethod, vctx ValidationContext) {
	if p.HasRequestBody() {
		rctx := vctx.At("request").atResource("request")
		rctx.checkResource()
		rctx.checkSchemaName(p.RequestBody)
	}

	walkResponses(p.Responses, vctx)
}

func walkDelete(d dsl.DeleteMethod, vctx ValidationContext) {
	walkResponses(d.Responses, vctx)
}

func walkResponses(r dsl.Responses, vctx ValidationContext) {
	vctx.checkResponses(r)

	for _, statusCode := range slices.Sorted(maps.Keys(r)) {
		sctx := vctx.At(strconv.Itoa(statusCode)).atResource("responses", strconv.Itoa(statusCode))
		sctx.checkResource()
		sctx.checkSchemaName(r[statusCode])
	}
}

// walkSchemas visits every declared schema, referenced by an endpoint or not, each one
// the root of its own descent and of its own depth budget.
func walkSchemas(s dsl.SchemasMap, vctx ValidationContext) {
	vctx.checkSchemas(s)

	for _, name := range slices.Sorted(maps.Keys(s)) {
		walkSchema(s[name], vctx.AtSchema(name))
	}
}

// walkSchema descends the schema, following its refs through the namespace. Following
// a ref is what catches a cycle: the descent only ends when the depth budget runs out.
func walkSchema(s dsl.Schema, vctx ValidationContext) {
	vctx.checkSchema(s)

	if vctx.Pos.Depth.Exceeded() {
		return
	}

	switch {
	case s.IsRef():
		target, declared := vctx.Index.Schema(s.Ref)
		if !declared {
			return
		}

		walkSchema(target, vctx.Deeper())

	case s.IsArray():
		if s.Items == nil {
			return
		}

		walkSchema(*s.Items, vctx.Deeper().AtItems())

	case s.IsObject():
		for _, name := range slices.Sorted(maps.Keys(s.Properties)) {
			walkSchema(s.Properties[name], vctx.Deeper().AtProperty(name))
		}
	}
}
