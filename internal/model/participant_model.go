package model

import "strings"

type Participant struct {
	ID   int64
	Name string
}

// ParticipantNameViolation judges the spelling of a participant name. The name is
// identity-bearing — a consumed service name matches a participant by it — so only
// snake_case is accepted: lowercase words of letters and digits joined by single
// underscores, nothing else.
func ParticipantNameViolation(name string) string {
	for _, word := range strings.Split(name, "_") {
		if word == "" {
			return "must be snake_case"
		}

		for _, character := range word {
			if (character < 'a' || character > 'z') && (character < '0' || character > '9') {
				return "must be snake_case"
			}
		}
	}

	return ""
}

func NewParticipant(name string) *Participant {
	return &Participant{
		Name: name,
	}
}
