package rename_participant

import (
	"context"
	"fmt"
	"time"

	"github.com/contracttesting/cli/internal/ui"
	"github.com/spf13/cobra"
)

const requestTimeout = 30 * time.Second

func NewRenameParticipantCommand(client *RenameParticipantClient) *cobra.Command {
	commandHandler := func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		oldName := args[0]
		newName := args[1]
		if oldName == "" || newName == "" {
			return fmt.Errorf("participant names must not be empty")
		}

		ctx, cancel := context.WithTimeout(command.Context(), requestTimeout)
		defer cancel()

		input := &RenameParticipantInput{OldName: oldName, NewName: newName}

		message, err := client.Rename(ctx, input)
		if err != nil {
			return err
		}

		ui.Success(command.OutOrStdout(), "✏️", message)
		return nil
	}

	command := &cobra.Command{
		Use:   "rename-participant [old] [new]",
		Short: "Rename an existing participant on the broker",
		Args:  cobra.ExactArgs(2),
		RunE:  commandHandler,
	}

	return command
}
