package dsl

type endpointSyntaxRule struct{}

func (endpointSyntaxRule) Code() string { return "endpoint.syntax" }

// The identity under judgment is the normalized spelling — a trailing slash is spelling
// without meaning — but the message quotes the raw one the user wrote.
func (endpointSyntaxRule) CheckEndpoint(endpoint string, vctx ValidationContext) {
	if reason := endpointViolation(normalizeEndpoint(endpoint)); reason != "" {
		vctx.Errs.Addf("invalid endpoint %q: %s (%s)", endpoint, reason, vctx.Pos.Source)
	}
}
