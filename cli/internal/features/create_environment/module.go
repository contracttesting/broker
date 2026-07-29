package create_environment

import (
	"github.com/contracttesting/cli/internal/components"
	"github.com/spf13/cobra"
)

func Register(rootCommand *cobra.Command, components *components.Components) {
	client := NewCreateEnvironmentClient(components.HTTPClient)
	command := NewCreateEnvironmentCommand(client)
	rootCommand.AddCommand(command)
}
