package publish_contract

import (
	"github.com/contracttesting/broker/internal/components"
	"github.com/contracttesting/broker/internal/features/publish_contract/validator"
	"github.com/contracttesting/broker/internal/repository"
)

func Register(components *components.Components) {
	contractRepository := repository.NewContractRepository(components.Pool)
	participantRepository := repository.NewParticipantRepository(components.Pool)
	contractValidator := validator.NewDslValidator()
	handler := NewPublishContractHandler(contractRepository, participantRepository, contractValidator)
	components.Server.Post("/api/contracts", handler.Handle)
}
