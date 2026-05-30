package create_environment

type CreateEnvironmentInput struct {
	Name string `json:"name"`
}

type CreateEnvironmentResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
