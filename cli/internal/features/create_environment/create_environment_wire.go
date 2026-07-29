package create_environment

type CreateEnvironmentRequestBody struct {
	Participant string `json:"participant"`
}

type CreateEnvironmentResponseBody struct {
	Message string `json:"message"`
}
