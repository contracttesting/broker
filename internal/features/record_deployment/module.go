package record_deployment

import (
	"github.com/contracttesting/cli/internal/components"
	"github.com/spf13/cobra"
)

func Register(rootCommand *cobra.Command, components *components.Components) {
	client := NewRecordDeploymentClient(components.HTTPClient)
	command := NewRecordDeploymentCommand(client)
	rootCommand.AddCommand(command)
}
