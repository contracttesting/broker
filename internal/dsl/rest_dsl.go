package dsl

import (
	"maps"
	"slices"
	"strconv"
)

type Rest map[string]HttpMethods

func (r Rest) Validate(vctx ValidationContext) {
	vctx.checkRest(r)

	visited := make(map[string]bool)

	for _, endpoint := range slices.Sorted(maps.Keys(r)) {
		vctx.checkEndpoint(endpoint)

		normalized := normalizeEndpoint(endpoint)

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

		r[endpoint].Validate(vctx.AtEndpoint(normalized).atResource("rest", normalized))
	}
}

type HttpMethods struct {
	Get    GetMethod    `json:"get,omitzero"`
	Post   PostMethod   `json:"post,omitzero"`
	Put    PutMethod    `json:"put,omitzero"`
	Delete DeleteMethod `json:"delete,omitzero"`
}

func (m HttpMethods) Validate(vctx ValidationContext) {
	m.Get.Validate(vctx.At("GET").atResource("get"))
	m.Post.Validate(vctx.At("POST").atResource("post"))
	m.Put.Validate(vctx.At("PUT").atResource("put"))
	m.Delete.Validate(vctx.At("DELETE").atResource("delete"))
}

type GetMethod struct {
	Responses Responses `json:"responses,omitzero"`
}

func (g GetMethod) Validate(vctx ValidationContext) {
	g.Responses.Validate(vctx)
}

func (g *GetMethod) IsNonZero() bool {
	return len(g.Responses) > 0
}

type PostMethod struct {
	RequestBody string    `json:"request,omitzero"`
	Responses   Responses `json:"responses,omitzero"`
}

func (p *PostMethod) HasRequestBody() bool {
	return p.RequestBody != ""
}

func (p *PostMethod) IsNonZero() bool {
	return p.RequestBody != "" || len(p.Responses) > 0
}

func (p PostMethod) Validate(vctx ValidationContext) {
	if p.HasRequestBody() {
		rctx := vctx.At("request").atResource("request")
		rctx.checkResource()
		rctx.checkSchemaName(p.RequestBody)
	}

	p.Responses.Validate(vctx)
}

type PutMethod struct {
	RequestBody string    `json:"request,omitzero"`
	Responses   Responses `json:"responses,omitzero"`
}

func (p *PutMethod) HasRequestBody() bool {
	return p.RequestBody != ""
}

func (p *PutMethod) IsNonZero() bool {
	return p.RequestBody != "" || len(p.Responses) > 0
}

func (p PutMethod) Validate(vctx ValidationContext) {
	if p.HasRequestBody() {
		rctx := vctx.At("request").atResource("request")
		rctx.checkResource()
		rctx.checkSchemaName(p.RequestBody)
	}

	p.Responses.Validate(vctx)
}

type DeleteMethod struct {
	Responses Responses `json:"responses,omitzero"`
}

func (d *DeleteMethod) IsNonZero() bool {
	return len(d.Responses) > 0
}

func (d DeleteMethod) Validate(vctx ValidationContext) {
	d.Responses.Validate(vctx)
}

const (
	MIN_STATUS_CODE = 100
	MAX_STATUS_CODE = 599
)

type Responses map[int]string

func (r Responses) Validate(vctx ValidationContext) {
	vctx.checkResponses(r)

	for _, statusCode := range slices.Sorted(maps.Keys(r)) {
		sctx := vctx.At(strconv.Itoa(statusCode)).atResource("responses", strconv.Itoa(statusCode))
		sctx.checkResource()
		sctx.checkSchemaName(r[statusCode])
	}
}
