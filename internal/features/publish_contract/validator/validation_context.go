package validator

type ValidationContext struct {
	Where      string
	RootSchema string
	Violations []string
	Depth      DepthCounter

	Source        string
	ContractIndex ContractIndex
	Rules         map[string][]Rule
}

func NewValidationContext(
	source string,
	contractIndex ContractIndex,
	rules map[string][]Rule,
) *ValidationContext {
	return &ValidationContext{
		Source:        source,
		ContractIndex: contractIndex,
		Rules:         rules,
		Depth:         DepthCounter{},
		Violations:    []string{},
	}
}

func (vc *ValidationContext) AddViolation(violation string) {
	vc.Violations = append(vc.Violations, violation)
}
