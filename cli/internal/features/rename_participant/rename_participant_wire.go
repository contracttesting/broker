package rename_participant

type RenameParticipantRequestBody struct {
	Name    string `json:"name"`
	NewName string `json:"newName"`
}

type RenameParticipantResponseBody struct {
	Message string `json:"message"`
}
