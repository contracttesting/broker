package rename_participant

import "github.com/contracttesting/broker/internal/shared"

const (
	ParticipantRenamed       string = "participant renamed"
	ParticipantInvalidInput  string = "participant invalid input"
	ParticipantAlreadyExists string = "participant already exists"
	ParticipantNotFound      string = "participant not found"
)

type RenameParticipantRequestBody struct {
	OldName string `json:"oldName"`
	NewName string `json:"newName"`
}

type RenameParticipantResponseBody struct {
	shared.BrokerResponseBody
}
