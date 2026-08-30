package integration_test

import (
	"context"
	"net/http"

	"github.com/gofiber/fiber/v3"
)

const recoveredParticipantBody = `{"participant":"recovered_service"}`

func (s *IntegrationSuite) TestPanicInHandler_Returns500AndServerKeepsServing() {
	s.Components.Server.Post("/__panic", func(fiber.Ctx) error {
		panic("handler exploded")
	})

	status, body := s.post("/__panic", `{}`)
	s.Equal(http.StatusInternalServerError, status)
	s.JSONEq(`{"message":"internal error"}`, body)

	status, body = s.post("/api/participants", recoveredParticipantBody)
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"message":"participant created"}`, body)
}

func (s *IntegrationSuite) TestSqlFailureInRepository_Returns500AndServerKeepsServing() {
	_, err := s.Pool.Exec(context.Background(), `ALTER TABLE participants RENAME TO participants_moved_away`)
	s.Require().NoError(err)

	defer func() {
		_, _ = s.Pool.Exec(context.Background(), `ALTER TABLE IF EXISTS participants_moved_away RENAME TO participants`)
	}()

	status, body := s.post("/api/participants", recoveredParticipantBody)
	s.Equal(http.StatusInternalServerError, status)
	s.JSONEq(`{"message":"internal error"}`, body)

	for _, internal := range []string{"participants", "SELECT", "INSERT", ".go", "pgx", "panic"} {
		s.NotContains(body, internal)
	}

	_, err = s.Pool.Exec(context.Background(), `ALTER TABLE participants_moved_away RENAME TO participants`)
	s.Require().NoError(err)

	status, body = s.post("/api/participants", recoveredParticipantBody)
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"message":"participant created"}`, body)
}
