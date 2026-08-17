package publish_contract

import (
	"github.com/contracttesting/broker/internal/components"
	"github.com/contracttesting/broker/internal/dsl"
	"github.com/contracttesting/broker/internal/repository"
)

func Register(components *components.Components) {
	contractRepository := repository.NewContractRepository(components.Pool)
	participantRepository := repository.NewParticipantRepository(components.Pool)
	// built once and reused by every request: the point where per-company rules will be
	// appended in the future
	validator := dsl.NewValidator()
	handler := NewPublishContractHandler(contractRepository, participantRepository, validator)
	components.Server.Post("/api/contracts", handler.Handle)
}
