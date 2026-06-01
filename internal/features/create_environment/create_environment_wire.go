package create_environment

import "github.com/contracttesting/broker/internal/shared"

const (
	EnvironmentCreated       string = "environment created"
	EnvironmentInvalidInput  string = "environment invalid input"
	EnvironmentAlreadyExists string = "environment already exists"
)

type CreateEnvironmentRequestBody struct {
	Participant string `json:"participant"`
}

type CreateEnvironmentResponseBody struct {
	shared.BrokerResponseBody
}
