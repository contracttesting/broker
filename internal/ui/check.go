package ui

import (
	"fmt"
	"io"
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

// Check writes the plain-text failure view: a headline, then one blank-line-
// separated block per counterpart, each listing its resources and their breaks
// bucketed under error-type sub-headers. No color, no icons, no summary.
func Check(w io.Writer, view CheckView) {
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
