package rename_participant

import (
	"github.com/bidirekt/broker/internal/components"
	"github.com/bidirekt/broker/internal/repository"
)

func Register(components *components.Components) {
	participantRepository := repository.NewParticipantRepository(components.Pool)
	handler := NewRenameParticipantHandler(participantRepository)
	components.Server.Post("/api/participants/rename", handler.Handle)
}
