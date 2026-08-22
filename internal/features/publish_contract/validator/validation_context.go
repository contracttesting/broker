package validator

type ValidationContext struct {
	Where         string
	RootSchema    string
	Violations    []string
	Depth         DepthCounter
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

func (vc *ValidationContext) Validate(segment string, value any) {
	for _, rule := range vc.Rules[segment] {
		rule.Validate(value, vc)
	}
}

func (vc *ValidationContext) AddViolation(violation string) {
	vc.Violations = append(vc.Violations, violation)
}
