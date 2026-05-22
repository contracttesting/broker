package upload_contract

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

const (
	defaultBrokerURL = "http://localhost:3000"
	requestTimeout   = 30 * time.Second
)

func newUploadCmd() *cobra.Command {
	var brokerURL string

	cmd := &cobra.Command{
		Use:   "upload <contract-file>",
		Short: "Upload a contract JSON file to the broker",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			contractJSON, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("read contract file: %w", err)
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), requestTimeout)
			defer cancel()

			result, err := NewClient(brokerURL).Upload(ctx, contractJSON)
			if err != nil {
				return err
			}

			switch {
			case result.Success != nil:
				fmt.Fprintln(cmd.OutOrStdout(), result.Success.Message)
				return nil
			case result.BreakingChanges != nil:
				fmt.Fprintln(cmd.ErrOrStderr(), result.BreakingChanges.Message)
				for _, item := range result.BreakingChanges.BreakingChanges {
					fmt.Fprintf(cmd.ErrOrStderr(),
						"  - %s %s %s %s (%s)\n",
						item.Resource.Direction,
						item.Resource.Method,
						item.Resource.Endpoint,
						item.Property,
						item.Reason,
					)
				}
				return fmt.Errorf("contract incompatible with stored counterparts")
			default:
				return fmt.Errorf("broker returned no recognizable result")
			}
		},
	}

	cmd.Flags().StringVar(&brokerURL, "broker-url", defaultBrokerURL, "Broker base URL")
	return cmd
}
