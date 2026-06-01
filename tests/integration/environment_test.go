package integration_test

import (
	"net/http"
)

const productionEnvironmentBody = `{"participant":"production"}`

func (s *IntegrationSuite) TestHappyPath_CreateEnvironment() {
	status, body := s.post("/api/environments", productionEnvironmentBody)
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"success":true,"message":"environment created"}`, body)

	s.Equal(1, s.countRows("environments"))
}

func (s *IntegrationSuite) TestUnhappyPath_DuplicateEnvironmentName() {
	status, _ := s.post("/api/environments", productionEnvironmentBody)
	s.Equal(http.StatusOK, status)

	status, body := s.post("/api/environments", productionEnvironmentBody)
	s.Equal(http.StatusBadRequest, status)
	s.JSONEq(`{"success":false,"message":"environment already exists"}`, body)

	s.Equal(1, s.countRows("environments"))
}

func (s *IntegrationSuite) TestUnhappyPath_MissingEnvironmentName() {
	status, body := s.post("/api/environments", `{}`)
	s.Equal(http.StatusBadRequest, status)
	s.JSONEq(`{"success":false,"message":"environment invalid input"}`, body)

	s.Equal(0, s.countRows("environments"))
}
