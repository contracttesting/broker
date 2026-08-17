package dsl

import (
	"maps"
	"slices"
	"strconv"
)

type Rest map[string]HttpMethods

func (r Rest) Validate(vctx ValidationContext) {
	for _, endpoint := range slices.Sorted(maps.Keys(r)) {
		if err := validateEndpoint(endpoint); err != nil {
			vctx.Errs.Addf("%s (%s)", err, vctx.Pos.Source)

			// what an invalid endpoint declares is unreachable anyway
			continue
		}

		r[endpoint].Validate(vctx.AtEndpoint(endpoint))
	}
}

type HttpMethods struct {
	Get    GetMethod    `json:"get,omitzero"`
	Post   PostMethod   `json:"post,omitzero"`
	Put    PutMethod    `json:"put,omitzero"`
	Delete DeleteMethod `json:"delete,omitzero"`
}

func (m HttpMethods) Validate(vctx ValidationContext) {
	m.Get.Validate(vctx.At("GET"))
	m.Post.Validate(vctx.At("POST"))
	m.Put.Validate(vctx.At("PUT"))
	m.Delete.Validate(vctx.At("DELETE"))
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
		validateSchemaName(vctx.At("request"), p.RequestBody)
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
		validateSchemaName(vctx.At("request"), p.RequestBody)
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

type Responses map[int]string

func (r Responses) Validate(vctx ValidationContext) {
	for _, statusCode := range slices.Sorted(maps.Keys(r)) {
		validateSchemaName(vctx.At(strconv.Itoa(statusCode)), r[statusCode])
	}
}
