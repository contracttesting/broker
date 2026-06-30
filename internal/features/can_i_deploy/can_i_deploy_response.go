package can_i_deploy

import (
	"sort"
	"strings"

	"github.com/contracttesting/broker/internal/compatibility_checker"
	"github.com/contracttesting/broker/internal/model"
	"github.com/guregu/null"
)

const (
	roleConsumer = "consumer"
	roleProvider = "provider"

	interactionRequest  = "request"
	interactionResponse = "response"

	detailKeyProperty = "property"
)

// edgeKey identifies one dependency edge: a consumer/provider pair plus which side is
// under check. null.String is comparable, so this struct is a valid map key.
type edgeKey struct {
	consumerName    string
	consumerVersion null.String
	providerName    string
	providerVersion null.String
	checked         string
}

// buildIncompatibilities reshapes the flat list of breaking changes detected by the
// compatibility checker into one incompatibility per dependency edge, lifting the break
// coordinates to the top level. It is a pure function: no database access. The checked
// side (the subject being deployed) takes its identity from the request; the counterpart
// side takes its deployed identity from the stored resource.
func buildIncompatibilities(
	report *compatibility_checker.CompatibilityReport,
	participant string,
	version string,
) []Incompatibility {
	grouped := map[edgeKey][]Break{}

	for _, breakingChange := range report.Breaks {
		key := edgeKeyOf(breakingChange, participant, version)
		grouped[key] = append(grouped[key], breakOf(breakingChange))
	}

	incompatibilities := make([]Incompatibility, 0, len(grouped))
	for key, breaks := range grouped {
		sortBreaks(breaks)
		incompatibilities = append(incompatibilities, Incompatibility{
			Consumer:      Party{Participant: key.consumerName, Version: key.consumerVersion, Role: roleConsumer, Checked: key.checked == roleConsumer},
			Provider:      Party{Participant: key.providerName, Version: key.providerVersion, Role: roleProvider, Checked: key.checked == roleProvider},
			BreakingCount: len(breaks),
			Breaks:        breaks,
		})
	}

	sortIncompatibilities(incompatibilities)

	return incompatibilities
}

// edgeKeyOf derives the dependency edge for a break. The checked side (the participant
// being deployed) takes the request's participant/version; the counterpart takes its
// resource's deployed identity, or null when there is no counterpart (resource-level
// reasons such as provider_resource_not_found).
func edgeKeyOf(b compatibility_checker.ContractBreakingChange, participant string, version string) edgeKey {
	checked := b.CheckedResource
	counterpart := b.CounterpartResource

	if checked.IsConsumer() {
		providerName := checked.ConsumedProvider.String
		providerVersion := null.String{}
		if counterpart != nil {
			providerName = counterpart.ParticipantName
			providerVersion = counterpart.ParticipantVersion
		}

		return edgeKey{
			consumerName:    participant,
			consumerVersion: null.StringFrom(version),
			providerName:    providerName,
			providerVersion: providerVersion,
			checked:         roleConsumer,
		}
	}

	return edgeKey{
		consumerName:    counterpart.ParticipantName,
		consumerVersion: counterpart.ParticipantVersion,
		providerName:    participant,
		providerVersion: null.StringFrom(version),
		checked:         roleProvider,
	}
}

// breakOf lifts a single break's coordinates to the top level, reading them from the
// checked resource (always present) and moving the property path out of details. The
// reason keeps its full name so it stays explicit about whether the break is property-
// level (property_*) or resource-level (provider_resource_*).
func breakOf(b compatibility_checker.ContractBreakingChange) Break {
	checked := b.CheckedResource

	return Break{
		Interaction: interactionOf(checked.Interaction),
		Method:      checked.Method,
		Endpoint:    checked.Endpoint,
		StatusCode:  checked.ResponseStatusCode,
		Property:    propertyOf(b.Details),
		Reason:      string(b.Reason),
		Details:     detailsWithoutProperty(b.Details),
	}
}

func interactionOf(interaction model.Interaction) string {
	if interaction == model.RestRequest {
		return interactionRequest
	}

	return interactionResponse
}

func propertyOf(details map[string]string) null.String {
	if property, ok := details[detailKeyProperty]; ok {
		return null.StringFrom(property)
	}

	return null.String{}
}

func detailsWithoutProperty(details map[string]string) map[string]string {
	filtered := make(map[string]string, len(details))
	for key, value := range details {
		if key == detailKeyProperty {
			continue
		}

		filtered[snakeToCamel(key)] = value
	}

	if len(filtered) == 0 {
		return nil
	}

	return filtered
}

// snakeToCamel converts a snake_case detail key (produced by the domain) into the
// camelCase the wire payload uses, e.g. "checked_property_type" -> "checkedPropertyType".
func snakeToCamel(key string) string {
	segments := strings.Split(key, "_")
	for i := 1; i < len(segments); i++ {
		if segments[i] == "" {
			continue
		}

		segments[i] = strings.ToUpper(segments[i][:1]) + segments[i][1:]
	}

	return strings.Join(segments, "")
}

func sortIncompatibilities(incompatibilities []Incompatibility) {
	sort.Slice(incompatibilities, func(i, j int) bool {
		a, b := incompatibilities[i], incompatibilities[j]

		if a.Consumer.Participant != b.Consumer.Participant {
			return a.Consumer.Participant < b.Consumer.Participant
		}
		if a.Provider.Participant != b.Provider.Participant {
			return a.Provider.Participant < b.Provider.Participant
		}
		if a.Consumer.Checked != b.Consumer.Checked {
			return a.Consumer.Checked
		}
		if a.Consumer.Version.ValueOrZero() != b.Consumer.Version.ValueOrZero() {
			return a.Consumer.Version.ValueOrZero() < b.Consumer.Version.ValueOrZero()
		}

		return a.Provider.Version.ValueOrZero() < b.Provider.Version.ValueOrZero()
	})
}

func sortBreaks(breaks []Break) {
	sort.Slice(breaks, func(i, j int) bool {
		a, b := breaks[i], breaks[j]

		if a.Interaction != b.Interaction {
			return interactionRank(a.Interaction) < interactionRank(b.Interaction)
		}
		if a.Endpoint != b.Endpoint {
			return a.Endpoint < b.Endpoint
		}
		if a.Method != b.Method {
			return a.Method < b.Method
		}
		if a.StatusCode.ValueOrZero() != b.StatusCode.ValueOrZero() {
			return a.StatusCode.ValueOrZero() < b.StatusCode.ValueOrZero()
		}

		return a.Property.ValueOrZero() < b.Property.ValueOrZero()
	})
}

func interactionRank(interaction string) int {
	if interaction == interactionRequest {
		return 0
	}

	return 1
}
