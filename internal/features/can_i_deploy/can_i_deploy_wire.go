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

func (c BreakingChange) Message(consumerName, providerName string) string {
	X, Y := consumerName, providerName
	P := c.Details["property"]
	T1 := c.Details["consumer_type"]
	T2 := c.Details["provider_type"]
	OP := c.operation()
	providerChecked := c.CheckedResource.Direction == "provides"

	switch c.Reason {
	case "missing_in_provider":
		if providerChecked {
			return "Provider " + Y + " no longer provides property " + P + " required by consumer " + X + " on " + OP
		}
		return "Consumer " + X + " requires property " + P + " but provider " + Y + " does not provide it on " + OP
	case "missing_in_consumer":
		if providerChecked {
			return "Provider " + Y + " now requires property " + P + ", not sent by consumer " + X + " on " + OP
		}
		return "Consumer " + X + " does not send required property " + P + " on " + OP
	case "type_mismatch":
		if providerChecked {
			return "Provider " + Y + " changed property " + P + " to " + T2 + "; consumer " + X + " expects " + T1 + " on " + OP
		}
		return "Consumer " + X + " expects " + P + " as " + T1 + " but provider " + Y + " provides " + T2 + " on " + OP
	case "optional_in_provider_required_in_consumer":
		if providerChecked {
			return "Provider " + Y + " made property " + P + " optional but consumer " + X + " requires it on " + OP
		}
		return "Consumer " + X + " requires property " + P + " but provider " + Y + " provides it as optional on " + OP
	case "optional_in_consumer_required_in_provider":
		if providerChecked {
			return "Provider " + Y + " now requires property " + P + ", sent as optional by consumer " + X + " on " + OP
		}
		return "Consumer " + X + " sends property " + P + " as optional but provider " + Y + " requires it on " + OP
	case "provider_resource_not_found":
		return "No " + OP + " was found"
	case "provider_resource_not_deployed_in_environment":
		if env, ok := c.Details["deployed_environments"]; ok {
			return OP + " exists but isn't deployed in this environment yet (deployed in: " + env + ")"
		}
		return OP + " exists but isn't deployed in any environment yet"
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
				providerName := change.CheckedResource.ConsumedProvider
				k := groupKey{counterpart: providerName, resource: change.resource()}
				dependsOn[k] = append(dependsOn[k], change.Message(key, providerName))
			} else {
				k := groupKey{counterpart: key, resource: change.resource()}
				dependedOnBy[k] = append(dependedOnBy[k], change.Message(key, participant))
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
