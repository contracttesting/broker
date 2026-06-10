package can_i_deploy

type CanIDeployRequestBody struct {
	Participant string `json:"participant"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
}

type BreakingResource struct {
	Direction          string  `json:"direction"`
	Kind               string  `json:"kind"`
	ConsumedProvider   *string `json:"consumed_provider"`
	Endpoint           string  `json:"endpoint"`
	Method             string  `json:"method"`
	ResponseStatusCode *string `json:"response_status_code"`
}

type BreakingChange struct {
	Reason              string            `json:"reason"`
	Details             map[string]string `json:"details"`
	CheckedResource     BreakingResource  `json:"checked_resource"`
	CounterpartResource BreakingResource  `json:"counterpart_resource"`
}

type CanIDeployResponseBody struct {
	Message    string                      `json:"message"`
	Deployable bool                        `json:"deployable"`
	Breaks     map[string][]BreakingChange `json:"breaks"`
}
