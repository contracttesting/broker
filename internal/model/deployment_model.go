package model

import "time"

type Deployment struct {
	ID          int64
	Participant *Participant
	Version     string
	Environment *Environment
	Rollback    bool
	Forced      bool
	DeployedAt  time.Time
}

func NewDeployment(participant *Participant, version string, environment *Environment, forced bool) *Deployment {
	return &Deployment{
		Participant: participant,
		Version:     version,
		Environment: environment,
		Forced:      forced,
	}
}
