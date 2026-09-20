package create_participant

import (
	"github.com/bidirekt/broker/internal/components"
	"github.com/bidirekt/broker/internal/repository"
)

func Register(components *components.Components) {
	repo := repository.NewParticipantRepository(components.Pool)
	handler := NewCreateParticipantHandler(repo)
	components.Server.Post("/api/participants", handler.Handle)
}
