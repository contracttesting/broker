package health

import (
	"github.com/contracttesting/broker/internal/components"
)

func Register(components *components.Components) {
	handler := NewHealthHandler()
	components.Server.Get("/health", handler.Handle)
}
