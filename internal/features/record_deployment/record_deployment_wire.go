package record_deployment

import "github.com/guregu/null"

const DeploymentRecorded string = "deployment recorded"
const DeploymentRecordedForced string = "deployment recorded despite a not deployable verdict"
const DeploymentInvalidInput string = "deployment invalid input"
const ParticipantNotFound string = "participant not found"
const VersionNotFound string = "version not found"
const EnvironmentNotFound string = "environment not found"

const ReasonCompatibilityCheckRequired string = "compatibility_check_required"
const ReasonNotDeployable string = "not_deployable"

type RecordDeploymentRequestBody struct {
	Participant string `json:"participant"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
	Force       bool   `json:"force"`
}

type RecordDeploymentResponseBody struct {
	Message string `json:"message"`
}

type CheckRequiredResult struct {
	CheckedVersion  null.String `json:"checkedVersion"`
	DeployedVersion null.String `json:"deployedVersion"`
}

type CheckRequiredResponseBody struct {
	Message string                         `json:"message"`
	Reason  string                         `json:"reason"`
	Results map[string]CheckRequiredResult `json:"results"`
}

type NotDeployableResult struct {
	CounterpartVersion null.String `json:"counterpartVersion"`
	Reason             string      `json:"reason"`
}

type NotDeployableResponseBody struct {
	Message string                         `json:"message"`
	Reason  string                         `json:"reason"`
	Results map[string]NotDeployableResult `json:"results"`
}
