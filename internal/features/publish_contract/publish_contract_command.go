package publish_contract

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/contracttesting/cli/pkg/multiparser"
	"github.com/spf13/cobra"
)

const requestTimeout = 30 * time.Second

func NewPublishCommand(publishContractClient *PublishContractClient) *cobra.Command {

	commandHandler := func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		filePath := args[0]

		if filePath == "" {
			return fmt.Errorf("no file path provided")
		}

		contractFileContent, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("read contract file: %w", err)
		}

		contractJSON, err := multiparser.AnyToJSON(args[0], contractFileContent)
		if err != nil {
			return err
		}

		participant, err := command.Flags().GetString("participant")
		if err != nil {
			return fmt.Errorf("get participant: %w", err)
		}

		version, err := command.Flags().GetString("version")
		if err != nil {
			return fmt.Errorf("get version: %w", err)
		}

		ctx, cancel := context.WithTimeout(command.Context(), requestTimeout)
		defer cancel()

		requestBody := &PublishContractRequestBody{
			Participant: participant,
			Version:     version,
			Contract:    contractJSON,
		}

		message, err := publishContractClient.PublishContract(ctx, requestBody)
		if err != nil {
			return err
		}

		fmt.Fprintf(command.OutOrStdout(), "📜 %s\n", message)

		return nil
	}

	command := &cobra.Command{
		Use:   "publish [file]",
		Short: "Publish a contract YAML or JSON file to the broker",
		Args:  cobra.ExactArgs(1),
		RunE:  commandHandler,
	}

	command.Flags().String("participant", "", "Participant name (required)")
	command.Flags().String("version", "", "Contract version, e.g. a commit hash or semver tag (required)")
	_ = command.MarkFlagRequired("participant")
	_ = command.MarkFlagRequired("version")

	return command
}
