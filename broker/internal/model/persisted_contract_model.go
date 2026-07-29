package model

type PersistedContract struct {
	ID              int64
	ParticipantID   int64
	ParticipantName string
	Version         string
	RawContract     string
	Resources       map[string]PersistedResource
}
