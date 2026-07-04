package can_i_deploy

type CanIDeployRequestBody struct {
	Participant string `json:"participant"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
}

type CanIDeployResponseBody struct {
	Message     string                      `json:"message"`
	Deployable  bool                        `json:"deployable"`
	Environment string                      `json:"environment"`
	Results     map[string]CanIDeployResult `json:"results"`
}

type CanIDeployResult struct {
	Deployable              bool                    `json:"deployable"`
	IncompatibleCounterpart IncompatibleCounterpart `json:"incompatibleCounterpart"`
	Breaks                  []ContractBreak         `json:"breaks"`
}

type IncompatibleCounterpart struct {
	ParticipantName    string  `json:"participantName"`
	ParticipantVersion *string `json:"participantVersion"`
}

type ContractBreak struct {
	CheckedResource     *BreakResource    `json:"checkedResource"`
	CounterpartResource *BreakResource    `json:"counterpartResource"`
	Reason              string            `json:"reason"`
	Details             map[string]string `json:"details"`
}

type BreakResource struct {
	Direction          string  `json:"direction"`
	Endpoint           string  `json:"endpoint"`
	Method             string  `json:"method"`
	ResponseStatusCode *string `json:"responseStatusCode"`
}
