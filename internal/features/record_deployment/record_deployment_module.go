package record_deployment

import (
	"github.com/bidirekt/broker/internal/components"
	"github.com/bidirekt/broker/internal/repository"
)

func Register(components *components.Components) {
	participantRepository := repository.NewParticipantRepository(components.Pool)
	contractRepository := repository.NewContractRepository(components.Pool)
	environmentRepository := repository.NewEnvironmentRepository(components.Pool)
	deploymentRepository := repository.NewDeploymentRepository(components.Pool)
	handler := NewRecordDeploymentHandler(deploymentRepository, participantRepository, contractRepository, environmentRepository)
	components.Server.Post("/api/deployments", handler.Handle)
}
