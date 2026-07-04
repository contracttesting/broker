package can_i_deploy

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const requestTimeout = 30 * time.Second

var ErrSilent = errors.New("failure already reported")

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
			fmt.Fprintf(command.ErrOrStderr(), "❌ %s\n", err.Error())
			return ErrSilent
		}

		if !resp.Deployable {
			fmt.Fprint(command.OutOrStdout(), formatNotDeployableReport(participant, environment, resp.Results))
			return ErrSilent
		}

		fmt.Fprintf(command.OutOrStdout(), "🚀 %s can be deployed to %s\n", participant, environment)

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

func formatNotDeployableReport(participant, environment string, results map[string]CanIDeployResult) string {
	var report strings.Builder
	fmt.Fprintf(&report, "❌ %s cannot be deployed to %s\n", participant, environment)

	counterparts := make([]string, 0, len(results))
	for name, result := range results {
		if !result.Deployable {
			counterparts = append(counterparts, name)
		}
	}
	sort.Strings(counterparts)

	for _, name := range counterparts {
		result := results[name]

		report.WriteString("\n" + name)
		if version := result.IncompatibleCounterpart.ParticipantVersion; version != nil {
			fmt.Fprintf(&report, " (%s)", *version)
		}
		report.WriteString(":\n")

		for _, contractBreak := range result.Breaks {
			fmt.Fprintf(&report, "  - %s\n", formatBreakLine(environment, contractBreak))
		}
	}

	return report.String()
}

func formatBreakLine(environment string, contractBreak ContractBreak) string {
	details := contractBreak.Details

	switch contractBreak.Reason {
	case "provider_resource_not_deployed_in_environment":
		line := fmt.Sprintf("provider is not deployed in %q", environment)
		if deployedIn, ok := details["deployedEnvironments"]; ok {
			line += fmt.Sprintf(" (deployed in: %s)", deployedIn)
		}
		return line
	case "provider_resource_not_found":
		return consumerResourcePrefix(contractBreak) + ": no matching resource in provider"
	case "property_missing_in_provider":
		return fmt.Sprintf("%s: property %q is missing in provider", consumerResourcePrefix(contractBreak), details["property"])
	case "property_missing_in_consumer":
		return fmt.Sprintf("%s: property %q is missing in consumer", consumerResourcePrefix(contractBreak), details["property"])
	case "property_optional_in_provider_required_in_consumer":
		return fmt.Sprintf("%s: property %q is optional in provider but required in consumer", consumerResourcePrefix(contractBreak), details["property"])
	case "property_optional_in_consumer_required_in_provider":
		return fmt.Sprintf("%s: property %q is optional in consumer but required in provider", consumerResourcePrefix(contractBreak), details["property"])
	case "property_type_mismatch":
		consumerType, providerType := details["checkedPropertyType"], details["counterpartPropertyType"]
		if contractBreak.CheckedResource != nil && contractBreak.CheckedResource.Direction != "consumes" {
			consumerType, providerType = providerType, consumerType
		}
		return fmt.Sprintf("%s: property %q type mismatch — consumer has %s, provider has %s",
			consumerResourcePrefix(contractBreak), details["property"], consumerType, providerType)
	default:
		return fallbackBreakLine(contractBreak.Reason, details)
	}
}

// consumerResourcePrefix formats "{METHOD} {endpoint} {interaction} {status}" (e.g.
// "GET /payments/* response 200", "POST /payments request") from the consumer-side
// resource, resolved by direction — never by checked/counterpart position.
func consumerResourcePrefix(contractBreak ContractBreak) string {
	resource := contractBreak.CheckedResource
	if resource == nil || resource.Direction != "consumes" {
		resource = contractBreak.CounterpartResource
	}

	parts := []string{strings.ToUpper(resource.Method), resource.Endpoint}
	if resource.Interaction != "" {
		parts = append(parts, strings.TrimPrefix(resource.Interaction, "rest_"))
	}
	if resource.ResponseStatusCode != nil {
		parts = append(parts, *resource.ResponseStatusCode)
	}

	return strings.Join(parts, " ")
}

// fallbackBreakLine renders an unknown reason code verbatim with its details, so the
// CLI never swallows a break it does not have a template for.
func fallbackBreakLine(reason string, details map[string]string) string {
	if len(details) == 0 {
		return reason
	}

	keys := make([]string, 0, len(details))
	for key := range details {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		pairs = append(pairs, key+": "+details[key])
	}

	return fmt.Sprintf("%s (%s)", reason, strings.Join(pairs, ", "))
}
