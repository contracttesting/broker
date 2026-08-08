package model

type Participant struct {
	ID   int64
	Name string
}

func NewParticipant(name string) *Participant {
	return &Participant{
		Name: name,
	}
}
