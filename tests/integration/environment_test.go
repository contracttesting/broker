package integration_test

import (
	"net/http"
)

const productionEnvironmentBody = `{"participant":"production"}`

func (s *IntegrationSuite) TestHappyPath_CreateEnvironment() {
	status, body := s.post("/api/environments", productionEnvironmentBody)
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"message":"environment created"}`, body)

	s.Equal(1, s.countRows("environments"))
}

func (s *IntegrationSuite) TestIdempotent_DuplicateEnvironmentName() {
	status, _ := s.post("/api/environments", productionEnvironmentBody)
	s.Equal(http.StatusOK, status)

	status, body := s.post("/api/environments", productionEnvironmentBody)
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"message":"environment already exists"}`, body)

	s.Equal(1, s.countRows("environments"))
}

func (s *IntegrationSuite) TestUnhappyPath_MissingEnvironmentName() {
	status, body := s.post("/api/environments", `{}`)
	s.Equal(http.StatusBadRequest, status)
	s.JSONEq(`{"message":"environment invalid input"}`, body)

	s.Equal(0, s.countRows("environments"))
}
