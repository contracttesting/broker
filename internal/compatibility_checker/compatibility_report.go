package compatibility_checker

import (
	"github.com/contracttesting/broker/internal/model"
	"github.com/guregu/null"
)

type IncompatibleCounterpart struct {
	ParticipantID      int64       `json:"-"`
	ParticipantName    string      `json:"participantName"`
	ParticipantVersion null.String `json:"participantVersion"`
}

type IncompatibleItem struct {
	Deployable              bool                     `json:"deployable"`
	IncompatibleCounterpart IncompatibleCounterpart  `json:"incompatibleCounterpart"`
	Breaks                  []ContractBreakingChange `json:"breaks"`
}

func NewIncompatibleItem() *IncompatibleItem {
	return &IncompatibleItem{
		Breaks: make([]ContractBreakingChange, 0),
	}
}

func (r *IncompatibleItem) AppendContractBreakChange(b ContractBreakingChange) {
	r.Breaks = append(r.Breaks, b)
	r.Deployable = false
}

type HierarchicalInteraction map[string][]ContractBreakingChange

type HierarchicalMethod map[string]HierarchicalInteraction

type HierarchicalEndpoint map[string]HierarchicalMethod

type Hierarchical struct {
	Deployable bool                 `json:"deployable"`
	Version    null.String          `json:"participantVersion"`
	Endpoints  HierarchicalEndpoint `json:"endpoints"`
}

type ContractCompatibilityReport struct {
	ParticipantName string
	Version         string
	Environment     string
	Results         map[string]IncompatibleItem
	Hierarchical    map[string]Hierarchical
}

func NewContractCompatibilityReport(
	participantName, version, environment string,
) *ContractCompatibilityReport {
	return &ContractCompatibilityReport{
		ParticipantName: participantName,
		Version:         version,
		Environment:     environment,
		Results:         make(map[string]IncompatibleItem),
		Hierarchical:    make(map[string]Hierarchical),
	}
}

func (r *ContractCompatibilityReport) AppendResult(dependency string, result *IncompatibleItem) {

	hierarchical, exists := r.Hierarchical[dependency]
	if !exists {
		hierarchical = Hierarchical{
			Deployable: true,
			Endpoints:  make(HierarchicalEndpoint),
		}
	}

	if !hierarchical.Version.Valid {
		hierarchical.Version = result.IncompatibleCounterpart.ParticipantVersion
	}

	for _, breakChange := range result.Breaks {
		endpointKey := breakChange.CheckedResource.Endpoint
		methodKey := breakChange.CheckedResource.Method

		var interactionKey string
		switch breakChange.CheckedResource.Interaction {
		case model.RestRequest:
			interactionKey = "request"
		case model.RestResponse:
			interactionKey = breakChange.CheckedResource.ResponseStatusCode.String
		default:
			interactionKey = breakChange.CheckedResource.Interaction.String()
		}

		if _, ok := hierarchical.Endpoints[endpointKey]; !ok {
			hierarchical.Endpoints[endpointKey] = make(HierarchicalMethod)
		}

		if _, ok := hierarchical.Endpoints[endpointKey][methodKey]; !ok {
			hierarchical.Endpoints[endpointKey][methodKey] = make(HierarchicalInteraction)
		}

		hierarchical.Endpoints[endpointKey][methodKey][interactionKey] = append(
			hierarchical.Endpoints[endpointKey][methodKey][interactionKey],
			breakChange,
		)

		hierarchical.Deployable = false
	}

	r.Hierarchical[dependency] = hierarchical

	existing := r.Results[dependency]

	if existing.Breaks == nil {
		existing.Breaks = []ContractBreakingChange{}
	}

	existing.Breaks = append(existing.Breaks, result.Breaks...)

	if existing.IncompatibleCounterpart.ParticipantID == 0 {
		existing.IncompatibleCounterpart.ParticipantID = result.IncompatibleCounterpart.ParticipantID
		existing.IncompatibleCounterpart.ParticipantVersion = result.IncompatibleCounterpart.ParticipantVersion
		existing.IncompatibleCounterpart.ParticipantName = dependency
	}

	existing.Deployable = len(existing.Breaks) == 0

	r.Results[dependency] = existing
}
