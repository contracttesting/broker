package create_participant

type CreateParticipantInput struct {
	Name string `json:"name"`
}

type CreateParticipantResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
