package integration_test

import (
	"net/http"
)

const petsParticipantBody = `{"participant":"pets_service"}`

func (s *IntegrationSuite) TestHappyPath_CreateParticipant() {
	status, body := s.post("/api/participants", petsParticipantBody)
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"message":"participant created"}`, body)

	s.Equal(1, s.countRows("participants"))
}

func (s *IntegrationSuite) TestCreateParticipant_NameNotSnakeCase_Rejected() {
	for _, name := range []string{"Pets-Service", "pets service", "PETS", "pets_"} {
		status, body := s.post("/api/participants", `{"participant":"`+name+`"}`)
		s.Equal(http.StatusBadRequest, status)
		s.JSONEq(`{"message":"participant name must be snake_case"}`, body)
	}

	s.Equal(0, s.countRows("participants"))
}

func (s *IntegrationSuite) TestIdempotent_DuplicateParticipantName() {
	status, _ := s.post("/api/participants", petsParticipantBody)
	s.Equal(http.StatusOK, status)

	status, body := s.post("/api/participants", petsParticipantBody)
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"message":"participant already exists"}`, body)

	s.Equal(1, s.countRows("participants"))
}
