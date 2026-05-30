package internal

import (
	"os"

	"github.com/contracttesting/cli/internal/components"
	"github.com/contracttesting/cli/internal/features/can_i_deploy"
	"github.com/contracttesting/cli/internal/features/create_environment"
	"github.com/contracttesting/cli/internal/features/create_participant"
	"github.com/contracttesting/cli/internal/features/publish_contract"
	"github.com/contracttesting/cli/internal/features/record_deployment"
	"github.com/contracttesting/cli/internal/features/rename_participant"
	"github.com/spf13/cobra"
)

var rootCommand = &cobra.Command{
	Use:   "ctio",
	Short: "CLI for ContractTesting",
}

func Run() {
	components := components.New()

	rootCommand.
		PersistentFlags().
		String(
			"broker-url",
			components.Config.BrokerURL,
			"Broker base URL",
		)

	// The HTTP client is built before flags are parsed, so apply --broker-url
	// (which defaults to the env/default URL) once Cobra has resolved it.
	rootCommand.PersistentPreRunE = func(command *cobra.Command, _ []string) error {
		brokerURL, err := command.Flags().GetString("broker-url")
		if err != nil {
			return err
		}
		components.HTTPClient.SetBaseURL(brokerURL)
		return nil
	}

	create_participant.Register(rootCommand, components)
	create_environment.Register(rootCommand, components)
	publish_contract.Register(rootCommand, components)
	record_deployment.Register(rootCommand, components)
	can_i_deploy.Register(rootCommand, components)
	rename_participant.Register(rootCommand, components)

	if err := rootCommand.Execute(); err != nil {
		os.Exit(1)
	}
}
