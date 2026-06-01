package record_deployment

import "github.com/contracttesting/cli/internal/shared"

type RecordDeploymentRequestBody struct {
	Participant string `json:"participant"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
}

type RecordDeploymentResponseBody struct {
	shared.BrokerResponseBody
}
