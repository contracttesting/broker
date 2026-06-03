package create_environment

import "github.com/contracttesting/cli/internal/shared"

type CreateEnvironmentRequestBody struct {
	Participant string `json:"participant"`
}

type CreateEnvironmentResponseBody struct {
	shared.BrokerResponseBody
}
