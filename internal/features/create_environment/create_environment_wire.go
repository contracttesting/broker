package create_environment

import "github.com/contracttesting/cli/internal/shared"

type CreateEnvironmentRequestBody struct {
	Name string `json:"name"`
}

type CreateEnvironmentResponseBody struct {
	shared.BrokerResponseBody
}
