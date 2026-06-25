package can_i_deploy

import "github.com/guregu/null"

const ContractNotFound = "contract not found"
const ParticipantNotFound = "participant not found"

type CanIDeployRequestBody struct {
	Participant string `json:"participant"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
}

type Party struct {
	Name    string      `json:"name"`
	Version null.String `json:"version"`
}

type Break struct {
	Interaction string            `json:"interaction"`
	Method      string            `json:"method"`
	Endpoint    string            `json:"endpoint"`
	StatusCode  null.String       `json:"statusCode"`
	Property    null.String       `json:"property"`
	Reason      string            `json:"reason"`
	Details     map[string]string `json:"details,omitempty"`
}

type Incompatibility struct {
	Consumer      Party   `json:"consumer"`
	Provider      Party   `json:"provider"`
	Checked       string  `json:"checked"`
	BreakingCount int     `json:"breakingCount"`
	Breaks        []Break `json:"breaks"`
}

type CanIDeployResponseBody struct {
	Message           string            `json:"message"`
	Deployable        bool              `json:"deployable"`
	Environment       string            `json:"environment"`
	Incompatibilities []Incompatibility `json:"incompatibilities"`
}

type CanIDeployErrorResponseBody struct {
	Message string `json:"message"`
}
