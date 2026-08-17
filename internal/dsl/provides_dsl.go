package dsl

type Provides struct {
	Rest    Rest              `json:"rest,omitzero"`
	Message map[string]string `json:"message,omitzero"`
}

// Message is a placeholder for messaging contracts: nothing downstream reads it, so the
// walk stops at Rest.
func (p Provides) Validate(vctx ValidationContext) {
	p.Rest.Validate(vctx)
}
