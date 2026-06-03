package can_i_deploy

import (
	"sort"
	"strings"

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
		Method           string `json:"method"`
		Endpoint         string `json:"endpoint"`
	} `json:"left_resource"`
}

func (c BreakingChange) resource() string {
	return strings.ToUpper(c.LeftResource.Method) + " " + c.LeftResource.Endpoint
}

type CanIDeployResponseBody struct {
	shared.BrokerResponseBody
	Deployable bool                        `json:"deployable"`
	Breaks     map[string][]BreakingChange `json:"breaks"`
}

type BreakGroup struct {
	Counterpart string
	Resource    string
	Messages    []string
}

type BreakSections struct {
	DependsOn    []BreakGroup
	DependedOnBy []BreakGroup
}

type groupKey struct {
	counterpart string
	resource    string
}

func (r CanIDeployResponseBody) GroupBreaks(participant string) BreakSections {
	dependsOn := map[groupKey][]string{}
	dependedOnBy := map[groupKey][]string{}

	for key, changes := range r.Breaks {
		for _, change := range changes {
			if key == participant {
				k := groupKey{counterpart: change.LeftResource.ConsumedProvider, resource: change.resource()}
				dependsOn[k] = append(dependsOn[k], change.HumanReadable)
			} else {
				k := groupKey{counterpart: key, resource: change.resource()}
				dependedOnBy[k] = append(dependedOnBy[k], change.HumanReadable)
			}
		}
	}

	return BreakSections{
		DependsOn:    sortedGroups(dependsOn),
		DependedOnBy: sortedGroups(dependedOnBy),
	}
}

func sortedGroups(grouped map[groupKey][]string) []BreakGroup {
	groups := make([]BreakGroup, 0, len(grouped))
	for key, messages := range grouped {
		sort.Strings(messages)
		groups = append(groups, BreakGroup{Counterpart: key.counterpart, Resource: key.resource, Messages: messages})
	}
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].Counterpart != groups[j].Counterpart {
			return groups[i].Counterpart < groups[j].Counterpart
		}
		return groups[i].Resource < groups[j].Resource
	})
	return groups
}
