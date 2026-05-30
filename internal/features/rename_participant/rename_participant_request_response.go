package rename_participant

type RenameParticipantInput struct {
	OldName string
	NewName string
}

type renameParticipantBody struct {
	Name    string `json:"name"`
	NewName string `json:"newName"`
}

type RenameParticipantResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
