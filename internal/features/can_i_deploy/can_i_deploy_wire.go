package can_i_deploy

import (
	"github.com/guregu/null"
)

const ContractNotFound = "contract not found"
const ParticipantNotFound = "participant not found"

type CanIDeployRequestBody struct {
	Participant string `json:"participant"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
}

type Party struct {
	Participant string      `json:"participant"`
	Version     null.String `json:"version"`
	Role        string      `json:"role"`
	Checked     bool        `json:"checked"`
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
	BreakingCount int     `json:"breakingCount"`
	Breaks        []Break `json:"breaks"`
}

type CanIDeployResponseBody struct {
	Message           string            `json:"message"`
	Participant       string            `json:"participant"`
	Version           string            `json:"version"`
	Environment       string            `json:"environment"`
	Deployable        bool              `json:"deployable"`
	Incompatibilities []Incompatibility `json:"incompatibilities"`
}

type CanIDeployErrorResponseBody struct {
	Message string `json:"message"`
}
