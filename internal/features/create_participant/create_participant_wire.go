package create_participant

const (
	ParticipantCreated          string = "participant created"
	ParticipantInvalidInput     string = "participant invalid input"
	ParticipantNameNotSnakeCase string = "participant name must be snake_case"
	ParticipantAlreadyExists    string = "participant already exists"
)

type CreateParticipantRequestBody struct {
	Participant string `json:"participant"`
}

type CreateParticipantResponseBody struct {
	Message string `json:"message"`
}
