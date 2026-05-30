package can_i_deploy

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

const requestTimeout = 30 * time.Second

func NewCanIDeployCommand(client *CanIDeployClient) *cobra.Command {
	commandHandler := func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true
		command.SilenceErrors = true

		participant := args[0]

		version, err := command.Flags().GetString("version")
		if err != nil {
			return fmt.Errorf("get version: %w", err)
		}

		environment, err := command.Flags().GetString("environment")
		if err != nil {
			return fmt.Errorf("get environment: %w", err)
		}

		ctx, cancel := context.WithTimeout(command.Context(), requestTimeout)
		defer cancel()

		input := &CanIDeployInput{
			Participant: participant,
			Version:     version,
			Environment: environment,
		}

		resp, err := client.Check(ctx, input)
		if err != nil {
			fmt.Fprintln(command.ErrOrStderr(), err)
			return err
		}

		if !resp.Deployable {
			fmt.Fprintln(command.OutOrStdout(), "not deployable")
			for _, message := range resp.BreakingChangeMessages() {
				fmt.Fprintf(command.OutOrStdout(), "  - %s\n", message)
			}
			return errNotDeployable
		}

		fmt.Fprintln(command.OutOrStdout(), "deployable")
		return nil
	}

	command := &cobra.Command{
		Use:   "can-i-deploy [participant]",
		Short: "Check whether a participant version can be deployed to an environment",
		Args:  cobra.ExactArgs(1),
		RunE:  commandHandler,
	}

	command.Flags().String("version", "", "Version to check, e.g. a commit hash or semver tag (required)")
	command.Flags().String("environment", "", "Target environment name (required)")
	_ = command.MarkFlagRequired("version")
	_ = command.MarkFlagRequired("environment")

	return command
}
