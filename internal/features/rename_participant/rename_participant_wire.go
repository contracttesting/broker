package rename_participant

import "github.com/contracttesting/cli/internal/shared"

type RenameParticipantRequestBody struct {
	Name    string `json:"name"`
	NewName string `json:"newName"`
}

type RenameParticipantResponseBody struct {
	shared.BrokerResponseBody
}
