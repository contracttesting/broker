package compatibility_checker

type CompatibilityResult struct {
	CounterpartParticipantID int64
	CounterpartVersion       string
	Deployable               bool
}

type CompatibilityReport struct {
	Results []CompatibilityResult       `json:"results"`
	Breaks  map[string][]BreakingChange `json:"breaks"`
}

func NewCompatibilityReport() *CompatibilityReport {
	return &CompatibilityReport{
		Results: make([]CompatibilityResult, 0),
		Breaks:  make(map[string][]BreakingChange),
	}
}

func (r *CompatibilityReport) Append(b BreakingChange) {
	if r.Breaks == nil {
		r.Breaks = make(map[string][]BreakingChange)
	}

	r.Breaks[b.ConsumerName()] = append(r.Breaks[b.ConsumerName()], b)
}
