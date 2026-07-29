package publish_contract

import (
	"github.com/contracttesting/cli/internal/components"
	"github.com/spf13/cobra"
)

func Register(rootCommand *cobra.Command, components *components.Components) {
	publishContractClient := NewPublishContractClient(components.HTTPClient)
	publishCommand := NewPublishCommand(publishContractClient)
	rootCommand.AddCommand(publishCommand)
}
