package can_i_deploy

import (
	"sort"

	"github.com/contracttesting/cli/internal/shared"
)

type CanIDeployRequestBody struct {
	Participant string `json:"participant"`
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

type CanIDeployResponseBody struct {
	shared.BrokerResponseBody
	Deployable bool                        `json:"deployable"`
	Breaks     map[string][]BreakingChange `json:"breaks"`
}

type BreakGroup struct {
	Counterpart string
	Messages    []string
}

type BreakSections struct {
	DependsOn    []BreakGroup
	DependedOnBy []BreakGroup
}

func (r CanIDeployResponseBody) GroupBreaks(participant string) BreakSections {
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
