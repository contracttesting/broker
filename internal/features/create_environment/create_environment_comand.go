package create_environment

import (
	"context"
	"fmt"
	"time"

	"github.com/contracttesting/cli/internal/ui"
	"github.com/spf13/cobra"
)

const requestTimeout = 30 * time.Second

func NewCreateEnvironmentCommand(client *CreateEnvironmentClient) *cobra.Command {
	commandHandler := func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		name := args[0]
		if name == "" {
			return fmt.Errorf("environment name must not be empty")
		}

		ctx, cancel := context.WithTimeout(command.Context(), requestTimeout)
		defer cancel()

		input := &CreateEnvironmentInput{Name: name}

		message, err := client.Create(ctx, input)
		if err != nil {
			return err
		}

		ui.Success(command.OutOrStdout(), "🌍", message)
		return nil
	}

	command := &cobra.Command{
		Use:   "create-environment [name]",
		Short: "Create a new environment on the broker",
		Args:  cobra.ExactArgs(1),
		RunE:  commandHandler,
	}

	return command
}
