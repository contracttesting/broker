package create_participant

import "github.com/contracttesting/cli/internal/shared"

type CreateParticipantRequestBody struct {
	Name string `json:"name"`
}

type CreateParticipantResponseBody struct {
	shared.BrokerResponseBody
}
