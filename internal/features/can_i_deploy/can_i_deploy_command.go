package can_i_deploy

import (
	"context"
	"fmt"
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

		input := &CanIDeployInput{
			Participant: participant,
			Version:     version,
			Environment: environment,
		}

		resp, err := client.Check(ctx, input)
		if err != nil {
			ui.Failure(command.ErrOrStderr(), err.Error())
			return ui.ErrSilent
		}

		if !resp.Deployable {
			out := command.OutOrStdout()
			ui.Failure(out, "not deployable")

			sections := resp.GroupBreaks(participant)
			if len(sections.DependsOn) > 0 {
				ui.GroupedTable(out,
					fmt.Sprintf("%s depends on these providers:", participant),
					[2]string{"PROVIDER", "BREAKING CHANGE"},
					tableGroups(sections.DependsOn))
			}
			if len(sections.DependedOnBy) > 0 {
				ui.GroupedTable(out,
					fmt.Sprintf("consumers that depend on %s:", participant),
					[2]string{"CONSUMER", "BREAKING CHANGE"},
					tableGroups(sections.DependedOnBy))
			}
			return ui.ErrSilent
		}

		ui.Success(command.OutOrStdout(), "🚀", "deployable")
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

func tableGroups(groups []BreakGroup) []ui.TableGroup {
	rows := make([]ui.TableGroup, len(groups))
	for i, group := range groups {
		rows[i] = ui.TableGroup{Label: group.Counterpart, Rows: group.Messages}
	}
	return rows
}
