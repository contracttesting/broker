package can_i_deploy

import "sort"

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
	LeftResource  struct {
		ConsumedProvider string `json:"consumed_provider"`
	} `json:"left_resource"`
}

type CanIDeployResponse struct {
	Success    bool                        `json:"success"`
	Deployable bool                        `json:"deployable"`
	Breaks     map[string][]BreakingChange `json:"breaks"`
}

// BreakGroup holds the breaking-change messages for one counterpart participant:
// a provider the checked participant depends on, or a consumer that depends on it.
type BreakGroup struct {
	Counterpart string
	Messages    []string
}

// BreakSections splits a not-deployable verdict by direction relative to the
// checked participant. A participant both provides and consumes, so a single
// check can yield breaks in both directions.
type BreakSections struct {
	DependsOn    []BreakGroup // providers the checked participant depends on
	DependedOnBy []BreakGroup // consumers that depend on the checked participant
}

// GroupBreaks groups the breaks for display. The broker keys breaks by the
// consumer participant, so breaks under the checked participant are the providers
// it depends on (grouped by each break's consumed provider); breaks under any
// other key are consumers that depend on it (grouped by that key). Groups and
// messages are sorted because the broker's map order is not stable.
func (r CanIDeployResponse) GroupBreaks(participant string) BreakSections {
	dependsOn := map[string][]string{}
	dependedOnBy := map[string][]string{}

	for key, changes := range r.Breaks {
		for _, change := range changes {
			if key == participant {
				provider := change.LeftResource.ConsumedProvider
				dependsOn[provider] = append(dependsOn[provider], change.HumanReadable)
			} else {
				dependedOnBy[key] = append(dependedOnBy[key], change.HumanReadable)
			}
		}
	}

	return BreakSections{
		DependsOn:    sortedGroups(dependsOn),
		DependedOnBy: sortedGroups(dependedOnBy),
	}
}

func sortedGroups(grouped map[string][]string) []BreakGroup {
	groups := make([]BreakGroup, 0, len(grouped))
	for counterpart, messages := range grouped {
		sort.Strings(messages)
		groups = append(groups, BreakGroup{Counterpart: counterpart, Messages: messages})
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Counterpart < groups[j].Counterpart })
	return groups
}
