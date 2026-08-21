package publish_contract

import (
	"github.com/contracttesting/broker/internal/components"
	"github.com/contracttesting/broker/internal/repository"
	"github.com/contracttesting/broker/internal/validator"
)

func Register(components *components.Components) {
	contractRepository := repository.NewContractRepository(components.Pool)
	participantRepository := repository.NewParticipantRepository(components.Pool)
	// built once and reused by every request: the point where per-company rules will be
	// appended in the future
	contractValidator := validator.New()
	handler := NewPublishContractHandler(contractRepository, participantRepository, contractValidator)
	components.Server.Post("/api/contracts", handler.Handle)
}
