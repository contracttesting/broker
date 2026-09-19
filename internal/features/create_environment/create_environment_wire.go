package create_environment

const (
	EnvironmentCreated       string = "environment created"
	EnvironmentInvalidInput  string = "environment invalid input"
	EnvironmentAlreadyExists string = "environment already exists"
)

type CreateEnvironmentRequestBody struct {
	Environment string `json:"environment"`
}

type CreateEnvironmentResponseBody struct {
	Message string `json:"message"`
}
