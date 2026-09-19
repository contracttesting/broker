package integration_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gofiber/fiber/v3"
)

func (s *IntegrationSuite) health(method string) (status int, response string) {
	req := httptest.NewRequest(method, "/health", nil)

	resp, err := s.Components.Server.Test(req, fiber.TestConfig{Timeout: 10 * time.Second})
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()

	bytes, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)

	return resp.StatusCode, string(bytes)
}

func (s *IntegrationSuite) TestHappyPath_HealthReportsStatusAndVersions() {
	status, body := s.health("GET")
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"status":"ok","brokerVersion":"dev","apiVersion":1}`, body)
}

func (s *IntegrationSuite) TestUnhappyPath_PostHealthIsNotRouted() {
	status, body := s.health("POST")
	s.Equal(http.StatusMethodNotAllowed, status)
	s.JSONEq(`{"message":"Method Not Allowed"}`, body)
}
