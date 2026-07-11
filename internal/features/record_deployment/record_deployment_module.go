package record_deployment

import (
	"github.com/contracttesting/broker/internal/compatibility_checker"
	"github.com/contracttesting/broker/internal/components"
	"github.com/contracttesting/broker/internal/repository"
)

func Register(components *components.Components) {
	participantRepository := repository.NewParticipantRepository(components.Pool)
	contractRepository := repository.NewContractRepository(components.Pool)
	environmentRepository := repository.NewEnvironmentRepository(components.Pool)
	deploymentRepository := repository.NewDeploymentRepository(components.Pool)
	compatibilityCheckRepository := repository.NewCompatibilityCheckRepository(components.Pool)
	compatibilityChecker := compatibility_checker.NewCompatibilityChecker(contractRepository)
	handler := NewRecordDeploymentHandler(deploymentRepository, participantRepository, contractRepository, environmentRepository, compatibilityCheckRepository, compatibilityChecker)
	components.Server.Post("/api/deployments", handler.Handle)
}
