package health

import (
	"github.com/bidirekt/broker/internal/components"
)

func Register(components *components.Components) {
	handler := NewHealthHandler()
	components.Server.Get("/health", handler.Handle)
}
