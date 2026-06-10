package can_i_deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/contracttesting/cli/internal/ui"
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

		requestBody := &CanIDeployRequestBody{
			Participant: participant,
			Version:     version,
			Environment: environment,
		}

		resp, err := client.Check(ctx, requestBody)
		if err != nil {
			ui.Failure(command.ErrOrStderr(), "❌", err.Error())
			return ui.ErrSilent
		}

		if !resp.Deployable {
			data, err := json.Marshal(resp.Breaks)
			if err != nil {
				return fmt.Errorf("marshal breaks: %w", err)
			}
			fmt.Println(string(data))
			os.Exit(1)
			return ui.ErrSilent
		}

		ui.Success(command.OutOrStdout(), "🚀", fmt.Sprintf("%s can be deployed to %s", participant, environment))

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
	command.MarkFlagRequired("version")
	command.MarkFlagRequired("environment")

	return command
}
