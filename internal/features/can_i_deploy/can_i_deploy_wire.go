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

func (c BreakingChange) resource() string {
	return strings.ToUpper(c.CheckedResource.Method) + " " + c.CheckedResource.Endpoint
}

func (c BreakingChange) operation() string {
	if c.CheckedResource.Kind == "rest_request" {
		return c.resource() + " (request)"
	}
	return c.resource() + " (response " + c.CheckedResource.ResponseStatusCode + ")"
}

func (c BreakingChange) Message() string {
	P := c.Details["property"]

	switch c.Reason {
	case "missing_in_provider":
		return P + ": required, not provided"
	case "missing_in_consumer":
		return P + ": required, not sent"
	case "type_mismatch":
		consumerType, providerType := c.Details["checked_property_type"], c.Details["counterpart_property_type"]
		if c.CheckedResource.Direction == "provides" {
			consumerType, providerType = providerType, consumerType
		}
		return P + ": consumer expects " + consumerType + ", provider provides " + providerType
	case "optional_in_provider_required_in_consumer":
		return P + ": required, provided as optional"
	case "optional_in_consumer_required_in_provider":
		return P + ": required, sent as optional"
	case "provider_resource_not_found":
		return "not found in provider"
	case "provider_resource_not_deployed_in_environment":
		if env, ok := c.Details["deployed_environments"]; ok {
			return "not deployed in this environment (deployed in: " + env + ")"
		}
		return "not deployed in any environment"
	default:
		return c.Reason
	}
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
				k := groupKey{counterpart: change.CheckedResource.ConsumedProvider, resource: change.operation()}
				dependsOn[k] = append(dependsOn[k], change.Message())
			} else {
				k := groupKey{counterpart: key, resource: change.operation()}
				dependedOnBy[k] = append(dependedOnBy[k], change.Message())
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
