package model

import (
	"time"

	"github.com/guregu/null"
)

type CompatibilityCheck struct {
	ID            int64
	ParticipantID int64
	Version       string
	EnvironmentID int64
	Deployable    bool
	CreatedAt     time.Time
	Results       []CompatibilityCheckResult
}

type CompatibilityCheckResult struct {
	ID                         int64
	CheckID                    int64
	CounterpartParticipantID   int64
	CounterpartParticipantName string
	CounterpartVersion         null.String
	Deployable                 bool
	Reason                     null.String
}
