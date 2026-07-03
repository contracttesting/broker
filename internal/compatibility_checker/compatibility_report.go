package compatibility_checker

import (
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

type ContractCompatibilityReport struct {
	ParticipantName string
	Version         string
	Environment     string
	Results         map[string]IncompatibleItem
}

func NewContractCompatibilityReport(
	participantName, version, environment string,
) *ContractCompatibilityReport {
	return &ContractCompatibilityReport{
		ParticipantName: participantName,
		Version:         version,
		Environment:     environment,
		Results:         make(map[string]IncompatibleItem),
	}
}

func (r *ContractCompatibilityReport) AppendResult(dependency string, result *IncompatibleItem) {
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
