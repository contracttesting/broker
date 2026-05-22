package upload_contract

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/spf13/cobra"
)

const requestTimeout = 30 * time.Second

func defaultBrokerURL() string {
	if u := os.Getenv("CONTRACTTESTING_BROKER_URL"); u != "" {
		return u
	}

	return "http://localhost:9000"
}

func newUploadCmd() *cobra.Command {
	var brokerURL string

	cmd := &cobra.Command{
		Use:   "upload <contract-file>",
		Short: "Upload a contract JSON file to the broker",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			raw, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("read contract file: %w", err)
			}

			contractJSON, err := toJSON(args[0], raw)
			if err != nil {
				return fmt.Errorf("encode contract as JSON: %w", err)
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

	cmd.Flags().StringVar(&brokerURL, "broker-url", defaultBrokerURL(), "Broker base URL")
	return cmd
}

func toJSON(path string, raw []byte) ([]byte, error) {
	switch ext := strings.ToLower(filepath.Ext(path)); ext {
	case ".json":
		return raw, nil
	case ".yaml", ".yml":
		return yaml.YAMLToJSON(raw)
	default:
		return nil, fmt.Errorf("unsupported contract file extension %q (want .json, .yaml, or .yml)", ext)
	}
}
