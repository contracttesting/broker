package can_i_deploy

import "github.com/contracttesting/broker/internal/compatibility_checker"

const ContractNotFound = "contract not found"
const ParticipantNotFound = "participant not found"

type CanIDeployRequestBody struct {
	Participant string `json:"participant"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
}

type CanIDeployResponseBody struct {
	Message    string                                            `json:"message"`
	Deployable bool                                              `json:"deployable"`
	Breaks     map[string][]compatibility_checker.BreakingChange `json:"breaks,omitempty"`
}

type CanIDeployErrorResponseBody struct {
	Message string `json:"message"`
}
