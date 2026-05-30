package can_i_deploy

import (
	"errors"
	"sort"
)

type CanIDeployInput struct {
	Participant string
	Version     string
	Environment string
}

type canIDeployBody struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
}

type BreakingChange struct {
	Reason        string `json:"reason"`
	Property      string `json:"property"`
	HumanReadable string `json:"human_readable"`
}

type CanIDeployResponse struct {
	Success    bool                        `json:"success"`
	Deployable bool                        `json:"deployable"`
	Breaks     map[string][]BreakingChange `json:"breaks"`
}

// BreakingChangeMessages returns every breaking change's human-readable text,
// sorted, so the command prints reasons deterministically: the broker groups
// breaks in a map and ranges a property map, so neither order is stable.
func (r CanIDeployResponse) BreakingChangeMessages() []string {
	var messages []string
	for _, changes := range r.Breaks {
		for _, change := range changes {
			messages = append(messages, change.HumanReadable)
		}
	}
	sort.Strings(messages)
	return messages
}

// errNotDeployable is returned solely to drive a non-zero exit when the broker
// answers "not deployable". The verdict is already on stdout, so the command
// sets SilenceErrors to keep this sentinel off the terminal.
var errNotDeployable = errors.New("not deployable")
