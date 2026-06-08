package can_i_deploy

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// CheckView is the failure view of one can-i-deploy check: a participant that
// cannot deploy to an environment, with the counterparts it breaks against.
type CheckView struct {
	Participant  string
	Environment  string
	Counterparts []Counterpart
}

// Counterpart is one incompatible counterpart and the resources breaking
// against it.
type Counterpart struct {
	Name      string
	Resources []Resource
}

// Resource is one failing interaction: an endpoint at a location ("request" or
// "{code} response") with its breaks bucketed by error type.
type Resource struct {
	Method, Path, Location string
	Groups                 []BreakGroup
}

// BreakGroup is one error-type bucket of breaks. Label is the "* " sub-header
// (e.g. "absent fields"); an empty Label means ungrouped — its breaks print
// directly under the resource with no header.
type BreakGroup struct {
	Label  string
	Breaks []string
}

// CheckView adapts the broker's breaks into the display model printBreakdown
// renders. Each break is flattened into a row, the rows are sorted once into
// display order, then folded into the counterpart → resource → group tree.
func (r CanIDeployResponseBody) CheckView(participant, environment string) CheckView {
	var rows []breakRow
	for key, changes := range r.Breaks {
		for _, change := range changes {
			rows = append(rows, newBreakRow(change, key, participant))
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].less(rows[j]) })

	return CheckView{
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
func groupByCounterpart(rows []breakRow) []Counterpart {
	var counterparts []Counterpart
	for _, row := range rows {
		if len(counterparts) == 0 || counterparts[len(counterparts)-1].Name != row.counterpart {
			counterparts = append(counterparts, Counterpart{Name: row.counterpart})
		}
		counterpart := &counterparts[len(counterparts)-1]

		if len(counterpart.Resources) == 0 || !resourceMatches(counterpart.Resources[len(counterpart.Resources)-1], row) {
			counterpart.Resources = append(counterpart.Resources, Resource{Method: row.method, Path: row.path, Location: row.location})
		}
		resource := &counterpart.Resources[len(counterpart.Resources)-1]

		if len(resource.Groups) == 0 || resource.Groups[len(resource.Groups)-1].Label != row.label {
			resource.Groups = append(resource.Groups, BreakGroup{Label: row.label})
		}
		group := &resource.Groups[len(resource.Groups)-1]

		group.Breaks = append(group.Breaks, row.line)
	}
	return counterparts
}

// resourceMatches reports whether row belongs to the same resource as r.
func resourceMatches(r Resource, row breakRow) bool {
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

// printBreakdown writes the plain-text failure view: a headline, then one blank-
// line-separated block per counterpart, each listing its resources and their
// breaks bucketed under error-type sub-headers. No color, no icons, no summary.
func printBreakdown(w io.Writer, view CheckView) {
	fmt.Fprintf(w, "%q cannot be deployed in %q environment\n", view.Participant, view.Environment)
	for _, counterpart := range view.Counterparts {
		fmt.Fprintln(w) // blank line between counterpart blocks
		fmt.Fprintf(w, "%s is not compatible with %s\n", view.Participant, counterpart.Name)
		for _, resource := range counterpart.Resources {
			fmt.Fprintf(w, "  %s %s (%s)\n", resource.Method, resource.Path, resource.Location)
			for _, group := range resource.Groups {
				writeBreakGroup(w, group)
			}
		}
	}
}

// writeBreakGroup writes one bucket of breaks under a resource. A labelled group
// prints a "* " header with its breaks indented beneath it; an unlabelled group
// prints its breaks flush under the resource.
func writeBreakGroup(w io.Writer, group BreakGroup) {
	indent := "    "
	if group.Label != "" {
		fmt.Fprintf(w, "    * %s\n", group.Label)
		indent = "      "
	}
	for _, line := range group.Breaks {
		fmt.Fprintf(w, "%s- %s\n", indent, line)
	}
}
