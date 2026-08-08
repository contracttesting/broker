package model

import "time"

type CompatibilityMatrix struct {
	ID                       int64
	ParticipantID            int64
	Version                  string
	CounterpartParticipantID int64
	CounterpartVersion       string
	Deployable               bool
	CreatedAt                time.Time
}
