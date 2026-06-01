package record_deployment

import "github.com/contracttesting/broker/internal/shared"

const DeploymentRecorded string = "deployment recorded"
const DeploymentInvalidInput string = "deployment invalid input"
const ParticipantNotFound string = "participant not found"
const VersionNotFound string = "version not found"
const EnvironmentNotFound string = "environment not found"

type RecordDeploymentRequestBody struct {
	Participant string `json:"participant"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
}

type RecordDeploymentResponseBody struct {
	shared.BrokerResponseBody
}
