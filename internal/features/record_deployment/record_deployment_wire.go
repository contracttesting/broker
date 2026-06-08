package record_deployment

type RecordDeploymentRequestBody struct {
	Participant string `json:"participant"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
}

type RecordDeploymentResponseBody struct {
	Message string `json:"message"`
}
