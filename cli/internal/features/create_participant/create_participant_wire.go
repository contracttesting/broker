package create_participant

type CreateParticipantRequestBody struct {
	Participant string `json:"participant"`
}

type CreateParticipantResponseBody struct {
	Message string `json:"message"`
}
