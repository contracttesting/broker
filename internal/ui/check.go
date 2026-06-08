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

// Check writes the flat failure view: a headline, then one group per
// counterpart — a header line, its resources, and each resource's breaks
// bucketed under "* " error-type sub-headers (ungrouped breaks print flat).
// Plain text only: no color, no icons, no summary.
func Check(w io.Writer, v CheckView) {
	fmt.Fprintf(w, "%q cannot be deployed in %q environment\n", v.Participant, v.Environment)
	for _, cp := range v.Counterparts {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "%s is not compatible with %s\n", v.Participant, cp.Name)
		for _, r := range cp.Resources {
			fmt.Fprintf(w, "  %s %s (%s)\n", r.Method, r.Path, r.Location)
			for _, g := range r.Groups {
				if g.Label != "" {
					fmt.Fprintf(w, "    * %s\n", g.Label)
					for _, b := range g.Breaks {
						fmt.Fprintf(w, "      - %s\n", b)
					}
					continue
				}
				for _, b := range g.Breaks {
					fmt.Fprintf(w, "    - %s\n", b)
				}
			}
		}
	}
}
