package create_participant

import "github.com/contracttesting/cli/internal/shared"

type CreateParticipantRequestBody struct {
	Participant string `json:"participant"`
}

type CreateParticipantResponseBody struct {
	shared.BrokerResponseBody
}
