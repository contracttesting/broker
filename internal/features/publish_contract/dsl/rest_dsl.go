package dsl

type Rest map[string]HttpMethods

type HttpMethods struct {
	Get    GetMethod    `json:"get,omitzero"`
	Post   PostMethod   `json:"post,omitzero"`
	Put    PutMethod    `json:"put,omitzero"`
	Delete DeleteMethod `json:"delete,omitzero"`
}

type GetMethod struct {
	Responses Responses `json:"responses,omitzero"`
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

type DeleteMethod struct {
	Responses Responses `json:"responses,omitzero"`
}

func (d *DeleteMethod) IsNonZero() bool {
	return len(d.Responses) > 0
}

type Responses map[int]string
