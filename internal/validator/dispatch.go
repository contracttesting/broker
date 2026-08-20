package validator

func dispatch(validationContext *ValidationContext, segment string, value any) {
	for _, rule := range validationContext.Rules[segment] {
		rule.Validate(value, validationContext)
	}
}
