package create_participant

import (
	"context"
	"fmt"
	"time"

	"github.com/contracttesting/cli/internal/ui"
	"github.com/spf13/cobra"
)

const requestTimeout = 30 * time.Second

func NewCreateParticipantCommand(client *CreateParticipantClient) *cobra.Command {
	commandHandler := func(command *cobra.Command, args []string) error {
		name := args[0]
		if name == "" {
			return fmt.Errorf("participant name must not be empty")
		}

		ctx, cancel := context.WithTimeout(command.Context(), requestTimeout)
		defer cancel()

		requestBody := &CreateParticipantRequestBody{
			Name: name,
		}

		message, err := client.Create(ctx, requestBody)
		if err != nil {
			return err
		}

		ui.Success(command.OutOrStdout(), "🎭", message)

		return nil
	}

	command := &cobra.Command{
		Use:   "create-participant [name]",
		Short: "Create a new participant on the broker",
		Args:  cobra.ExactArgs(1),
		RunE:  commandHandler,
	}

	return command
}
