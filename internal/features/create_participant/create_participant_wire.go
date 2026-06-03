package create_participant

const (
	ParticipantCreated       string = "participant created"
	ParticipantInvalidInput  string = "participant invalid input"
	ParticipantAlreadyExists string = "participant already exists"
)

type CreateParticipantRequestBody struct {
	Participant string `json:"participant"`
}

type CreateParticipantResponseBody struct {
	Message string `json:"message"`
}
