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

// CheckView adapts the broker's breaks into the display model ui.Check renders.
// Each break is flattened into a row, the rows are sorted once into display
// order, then folded into the counterpart → resource → group tree.
func (r CanIDeployResponseBody) CheckView(participant, environment string) ui.CheckView {
	var rows []breakRow
	for key, changes := range r.Breaks {
		for _, change := range changes {
			rows = append(rows, newBreakRow(change, key, participant))
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].less(rows[j]) })

	return ui.CheckView{
		Participant:  participant,
		Environment:  environment,
		Counterparts: groupByCounterpart(rows),
	}
}

// breakRow is one breaking change flattened into the rendered line plus the keys
// it is sorted and grouped by: counterpart, then resource (method, path,
// location), then error-type label, then field name.
type breakRow struct {
	counterpart, method, path, location, label, property, line string
}

// newBreakRow flattens one breaking change against the checked participant.
func newBreakRow(change BreakingChange, key, participant string) breakRow {
	counterpart, counterpartIsProvider := counterpartOf(change, key, participant)
	return breakRow{
		counterpart: counterpart,
		method:      strings.ToUpper(change.CheckedResource.Method),
		path:        change.CheckedResource.Endpoint,
		location:    breakLocation(change),
		label:       groupLabel(change.Reason),
		property:    change.Details["property"],
		line:        breakLine(change, participant, counterpart, counterpartIsProvider),
	}
}

// less orders rows by the display hierarchy, so folding consecutive rows
// reproduces the nested layout: counterpart, resource, label, field, then line.
func (a breakRow) less(b breakRow) bool {
	switch {
	case a.counterpart != b.counterpart:
		return a.counterpart < b.counterpart
	case a.method != b.method:
		return a.method < b.method
	case a.path != b.path:
		return a.path < b.path
	case a.location != b.location:
		return a.location < b.location
	case a.label != b.label:
		return a.label < b.label
	case a.property != b.property:
		return a.property < b.property
	default:
		return a.line < b.line
	}
}

// counterpartOf returns the side opposite the participant in a break and whether
// it is the provider. The breaks map is keyed by the side under check; when that
// key is the participant itself, the counterpart is the consumed provider.
func counterpartOf(change BreakingChange, key, participant string) (name string, isProvider bool) {
	if key == participant {
		return change.CheckedResource.ConsumedProvider, true
	}
	return key, false
}

// breakLocation describes where a break sits: "request" for a request body, or
// "{code} response" for a response body.
func breakLocation(change BreakingChange) string {
	if change.CheckedResource.Kind == "rest_request" {
		return "request"
	}
	return change.CheckedResource.ResponseStatusCode + " response"
}

// groupByCounterpart folds pre-sorted rows into the counterpart → resource →
// group tree, opening a new node whenever its key changes from the previous row.
func groupByCounterpart(rows []breakRow) []ui.Counterpart {
	var counterparts []ui.Counterpart
	for _, row := range rows {
		if len(counterparts) == 0 || counterparts[len(counterparts)-1].Name != row.counterpart {
			counterparts = append(counterparts, ui.Counterpart{Name: row.counterpart})
		}
		counterpart := &counterparts[len(counterparts)-1]

		if len(counterpart.Resources) == 0 || !resourceMatches(counterpart.Resources[len(counterpart.Resources)-1], row) {
			counterpart.Resources = append(counterpart.Resources, ui.Resource{Method: row.method, Path: row.path, Location: row.location})
		}
		resource := &counterpart.Resources[len(counterpart.Resources)-1]

		if len(resource.Groups) == 0 || resource.Groups[len(resource.Groups)-1].Label != row.label {
			resource.Groups = append(resource.Groups, ui.BreakGroup{Label: row.label})
		}
		group := &resource.Groups[len(resource.Groups)-1]

		group.Breaks = append(group.Breaks, row.line)
	}
	return counterparts
}

// resourceMatches reports whether row belongs to the same resource as r.
func resourceMatches(r ui.Resource, row breakRow) bool {
	return r.Method == row.method && r.Path == row.path && r.Location == row.location
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

// breakLine renders one property mismatch counterpart-first:
// "{property}: {counterpart state} in {counterpart} - {participant state} in
// {participant}". Each side's state comes from the broker reason; the
// counterpart's state goes left. Non-property (resource-level) reasons return
// the raw reason verbatim.
func breakLine(change BreakingChange, participant, counterpart string, counterpartIsProvider bool) string {
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
	if counterpartIsProvider {
		counterpartState, participantState = providerState, consumerState
	}
	return fmt.Sprintf("%s: %s in %s - %s in %s", change.Details["property"], counterpartState, counterpart, participantState, participant)
}
