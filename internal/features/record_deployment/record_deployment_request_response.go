package record_deployment

type RecordDeploymentInput struct {
	Participant string
	Version     string
	Environment string
}

type recordDeploymentBody struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
}

type RecordDeploymentResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
