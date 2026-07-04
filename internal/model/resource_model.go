package model

const (
	Consumes     Direction   = "consumes"
	Provides     Direction   = "provides"
	RestRequest  Interaction = "rest_request"
	RestResponse Interaction = "rest_response"
)

type Direction string

func (direction Direction) String() string {
	return string(direction)
}

type Interaction string

func (interaction Interaction) String() string {
	return string(interaction)
}
