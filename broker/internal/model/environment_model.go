package model

type Environment struct {
	ID   int64
	Name string
}

func NewEnvironment(name string) *Environment {
	return &Environment{
		Name: name,
	}
}
