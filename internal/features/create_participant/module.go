package create_participant

import (
	"github.com/contracttesting/cli/internal/components"
	"github.com/spf13/cobra"
)

func Register(rootCommand *cobra.Command, components *components.Components) {
	client := NewCreateParticipantClient(components.HTTPClient)
	command := NewCreateParticipantCommand(client)
	rootCommand.AddCommand(command)
}
