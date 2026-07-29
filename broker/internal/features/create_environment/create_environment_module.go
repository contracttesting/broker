package create_environment

import (
	"github.com/contracttesting/broker/internal/components"
	"github.com/contracttesting/broker/internal/repository"
)

func Register(components *components.Components) {
	repo := repository.NewEnvironmentRepository(components.Pool)
	handler := NewCreateEnvironmentHandler(repo)
	components.Server.Post("/api/environments", handler.Handle)
}
