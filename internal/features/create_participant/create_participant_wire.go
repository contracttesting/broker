package create_participant

import "github.com/contracttesting/broker/internal/shared"

const (
	ParticipantCreated       string = "participant created"
	ParticipantInvalidInput  string = "participant invalid input"
	ParticipantAlreadyExists string = "participant already exists"
)

type CreateParticipantRequestBody struct {
	Participant string `json:"participant"`
}

type CreateParticipantResponseBody struct {
	shared.BrokerResponseBody
}
