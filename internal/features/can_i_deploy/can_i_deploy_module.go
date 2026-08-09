package can_i_deploy

import (
	"github.com/contracttesting/broker/internal/compatibility_checker"
	"github.com/contracttesting/broker/internal/components"
	"github.com/contracttesting/broker/internal/repository"
)

func Register(components *components.Components) {
	contractRepository := repository.NewContractRepository(components.Pool)
	participantRepository := repository.NewParticipantRepository(components.Pool)
	compatibilityRepository := repository.NewCompatibilityRepository(components.Pool)
	compatibilityChecker := compatibility_checker.NewCompatibilityChecker(
		contractRepository,
		compatibilityRepository,
	)
	environmentRepository := repository.NewEnvironmentRepository(components.Pool)

	handler := NewCanIDeployHandler(
		contractRepository,
		environmentRepository,
		compatibilityRepository,
		compatibilityChecker,
		participantRepository,
	)

	components.Server.Post("/api/can-i-deploy", handler.Handle)
}
