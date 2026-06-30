package compatibility_checker

type CompatibilityResult struct {
	CounterpartParticipantID int64
	CounterpartVersion       string
	Deployable               bool
}

type CompatibilityReport struct {
	Results map[string]CompatibilityResult `json:"results"`
	Breaks  []ContractBreakingChange       `json:"breaks"`
}

func NewCompatibilityReport() *CompatibilityReport {
	return &CompatibilityReport{
		Results: make(map[string]CompatibilityResult),
		Breaks:  []ContractBreakingChange{},
	}
}

func (r *CompatibilityReport) AppendResult(dependency string, result CompatibilityResult) {
	if r.Results == nil {
		r.Results = make(map[string]CompatibilityResult)
	}

	existing, seen := r.Results[dependency]
	if !seen {
		r.Results[dependency] = result
		return
	}

	existing.Deployable = existing.Deployable && result.Deployable
	if existing.CounterpartParticipantID == 0 {
		existing.CounterpartParticipantID = result.CounterpartParticipantID
		existing.CounterpartVersion = result.CounterpartVersion
	}
	r.Results[dependency] = existing
}

func (r *CompatibilityReport) AppendContractBreakChange(b ContractBreakingChange) {
	r.Breaks = append(r.Breaks, b)
}
