package can_i_deploy

import (
	"sort"
	"strings"

	"github.com/contracttesting/cli/internal/shared"
	"github.com/contracttesting/cli/internal/ui"
)

type CanIDeployRequestBody struct {
	Participant string `json:"participant"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
}

type BreakingChange struct {
	Reason          string            `json:"reason"`
	Details         map[string]string `json:"details"`
	CheckedResource struct {
		Direction          string `json:"direction"`
		Kind               string `json:"kind"`
		ConsumedProvider   string `json:"consumed_provider"`
		Endpoint           string `json:"endpoint"`
		Method             string `json:"method"`
		ResponseStatusCode string `json:"response_status_code"`
	} `json:"checked_resource"`
}

type CanIDeployResponseBody struct {
	shared.BrokerResponseBody
	Deployable bool                        `json:"deployable"`
	Breaks     map[string][]BreakingChange `json:"breaks"`
}

// CheckView maps the breaks into the ui failure view. The breaks map is keyed
// by consumer name: the checked participant's own key holds its consumer-side
// breaks (counterpart = the consumed provider) and every other key is a
// consumer broken by the checked participant's provider side (counterpart =
// that consumer). Properties arrive in the broker's $.-prefixed dialect and
// are rendered verbatim. Map iteration order is random, so integrations sort
// by identity and mismatches by property.
func (r CanIDeployResponseBody) CheckView(participant, environment string) ui.CheckView {
	type identity struct{ counterpart, method, path, location string }
	integrations := map[identity]*ui.Integration{}

	for key, changes := range r.Breaks {
		for _, change := range changes {
			consumer, provider := key, participant
			if key == participant {
				provider = change.CheckedResource.ConsumedProvider
			}
			counterpart := consumer
			if consumer == participant {
				counterpart = provider
			}

			request := change.CheckedResource.Kind == "rest_request"
			location := "request"
			if !request {
				location = "response " + change.CheckedResource.ResponseStatusCode
			}

			id := identity{counterpart, strings.ToUpper(change.CheckedResource.Method), change.CheckedResource.Endpoint, location}
			integration, ok := integrations[id]
			if !ok {
				integration = &ui.Integration{Counterpart: id.counterpart, Method: id.method, Path: id.path, Location: id.location}
				integrations[id] = integration
			}

			property := change.Details["property"]

			switch change.Reason {
			case "missing_in_consumer":
				integration.Mismatches = append(integration.Mismatches, ui.Mismatch{Kind: ui.Missing, Property: property, Omitter: consumer, Requirer: provider})
			case "missing_in_provider":
				integration.Mismatches = append(integration.Mismatches, ui.Mismatch{Kind: ui.Missing, Property: property, Omitter: provider, Requirer: consumer})
			case "type_mismatch":
				consumerType, providerType := change.Details["checked_property_type"], change.Details["counterpart_property_type"]
				if change.CheckedResource.Direction == "provides" {
					consumerType, providerType = providerType, consumerType
				}
				wanter, wants, sender, sends := consumer, consumerType, provider, providerType
				if request {
					wanter, wants, sender, sends = provider, providerType, consumer, consumerType
				}
				integration.Mismatches = append(integration.Mismatches, ui.Mismatch{Kind: ui.TypeMismatch, Property: property, Wanter: wanter, Wants: wants, Sender: sender, Sends: sends})
			case "optional_in_consumer_required_in_provider":
				integration.Mismatches = append(integration.Mismatches, ui.Mismatch{Kind: ui.Optional, Property: property, Requirer: provider, Sender: consumer})
			case "optional_in_provider_required_in_consumer":
				integration.Mismatches = append(integration.Mismatches, ui.Mismatch{Kind: ui.Optional, Property: property, Requirer: consumer, Sender: provider})
			case "provider_resource_not_found":
				integration.Issues = append(integration.Issues, ui.Issue{Kind: ui.NotProvided})
			case "provider_resource_not_deployed_in_environment":
				integration.Issues = append(integration.Issues, ui.Issue{Kind: ui.NotDeployed, DeployedIn: change.Details["deployed_environments"]})
			default:
				integration.Issues = append(integration.Issues, ui.Issue{Kind: ui.UnknownReason, Reason: change.Reason})
			}
		}
	}

	view := ui.CheckView{Pacticipant: participant, Target: environment}
	for _, integration := range integrations {
		sort.Slice(integration.Mismatches, func(i, j int) bool { return integration.Mismatches[i].Property < integration.Mismatches[j].Property })
		view.Integrations = append(view.Integrations, *integration)
	}
	sort.Slice(view.Integrations, func(i, j int) bool {
		a, b := view.Integrations[i], view.Integrations[j]
		if a.Counterpart != b.Counterpart {
			return a.Counterpart < b.Counterpart
		}
		if a.Method != b.Method {
			return a.Method < b.Method
		}
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		return a.Location < b.Location
	})
	return view
}
