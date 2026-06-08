package can_i_deploy

import (
	"fmt"
	"sort"
	"strings"

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
	Message    string                      `json:"message"`
	Deployable bool                        `json:"deployable"`
	Breaks     map[string][]BreakingChange `json:"breaks"`
}

func (r CanIDeployResponseBody) CheckView(participant, environment string) ui.CheckView {
	type resourceKey struct{ method, path, location string }
	type breakItem struct{ property, line string }
	byCounterpart := map[string]map[resourceKey]map[string][]breakItem{}

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

			location := "request"
			if change.CheckedResource.Kind != "rest_request" {
				location = change.CheckedResource.ResponseStatusCode + " response"
			}

			resources, ok := byCounterpart[counterpart]
			if !ok {
				resources = map[resourceKey]map[string][]breakItem{}
				byCounterpart[counterpart] = resources
			}
			rk := resourceKey{strings.ToUpper(change.CheckedResource.Method), change.CheckedResource.Endpoint, location}
			groups, ok := resources[rk]
			if !ok {
				groups = map[string][]breakItem{}
				resources[rk] = groups
			}

			label := groupLabel(change.Reason)
			groups[label] = append(groups[label], breakItem{change.Details["property"], breakLine(change, participant, counterpart, provider)})
		}
	}

	view := ui.CheckView{Participant: participant, Environment: environment}
	for name, resources := range byCounterpart {
		cp := ui.Counterpart{Name: name}
		for rk, groups := range resources {
			resource := ui.Resource{Method: rk.method, Path: rk.path, Location: rk.location}
			for label, items := range groups {
				sort.Slice(items, func(i, j int) bool {
					if items[i].property != items[j].property {
						return items[i].property < items[j].property
					}
					return items[i].line < items[j].line
				})
				lines := make([]string, len(items))
				for i, item := range items {
					lines[i] = item.line
				}
				resource.Groups = append(resource.Groups, ui.BreakGroup{Label: label, Breaks: lines})
			}
			sort.Slice(resource.Groups, func(i, j int) bool {
				return resource.Groups[i].Label < resource.Groups[j].Label
			})
			cp.Resources = append(cp.Resources, resource)
		}
		sort.Slice(cp.Resources, func(i, j int) bool {
			a, b := cp.Resources[i], cp.Resources[j]
			if a.Method != b.Method {
				return a.Method < b.Method
			}
			if a.Path != b.Path {
				return a.Path < b.Path
			}
			return a.Location < b.Location
		})
		view.Counterparts = append(view.Counterparts, cp)
	}
	sort.Slice(view.Counterparts, func(i, j int) bool {
		return view.Counterparts[i].Name < view.Counterparts[j].Name
	})
	return view
}

// groupLabel maps a broker reason to its error-type sub-header. Field-level
// reasons bucket under a label; every other (resource-level) reason returns ""
// to print ungrouped.
func groupLabel(reason string) string {
	switch reason {
	case "missing_in_consumer", "missing_in_provider":
		return "absent fields"
	case "type_mismatch":
		return "type mismatches"
	case "optional_in_consumer_required_in_provider", "optional_in_provider_required_in_consumer":
		return "optional fields"
	default:
		return ""
	}
}

// breakLine renders one break counterpart-first. Property mismatches read
// "{property}: {counterpart state} in {counterpart} - {participant state} in
// {participant}", with each side's state derived from the broker reason; the
// counterpart's state goes left by whichever role (consumer/provider) it plays.
// Every other reason prints the raw broker reason verbatim.
func breakLine(change BreakingChange, participant, counterpart, provider string) string {
	var consumerState, providerState string
	switch change.Reason {
	case "missing_in_consumer":
		consumerState, providerState = "absent", "required"
	case "missing_in_provider":
		consumerState, providerState = "required", "absent"
	case "type_mismatch":
		consumerState, providerState = change.Details["checked_property_type"], change.Details["counterpart_property_type"]
		if change.CheckedResource.Direction == "provides" {
			consumerState, providerState = providerState, consumerState
		}
	case "optional_in_consumer_required_in_provider":
		consumerState, providerState = "optional", "required"
	case "optional_in_provider_required_in_consumer":
		consumerState, providerState = "required", "optional"
	default:
		return change.Reason
	}

	counterpartState, participantState := consumerState, providerState
	if counterpart == provider {
		counterpartState, participantState = providerState, consumerState
	}
	return fmt.Sprintf("%s: %s in %s - %s in %s", change.Details["property"], counterpartState, counterpart, participantState, participant)
}
