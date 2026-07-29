package can_i_deploy

import (
	"github.com/contracttesting/cli/internal/components"
	"github.com/spf13/cobra"
)

func Register(rootCommand *cobra.Command, components *components.Components) {
	client := NewCanIDeployClient(components.HTTPClient)
	command := NewCanIDeployCommand(client)
	rootCommand.AddCommand(command)
}
