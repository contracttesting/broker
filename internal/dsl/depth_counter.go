package dsl

import "fmt"

const MAX_DEPTH = 10

type DepthCounter struct {
	counter    int
	crashAt    int
	schemaName string
}

func (dc *DepthCounter) Enter() {
	dc.counter = dc.counter + 1

	if dc.counter >= dc.crashAt {
		panic(fmt.Sprintf(
			"schema %s is too deep with more than %d levels",
			dc.schemaName,
			dc.crashAt,
		))
	}
}

func (dc *DepthCounter) Exit() {
	dc.counter = dc.counter - 1
}

func NewDepthCounter(schemaName string) *DepthCounter {
	return &DepthCounter{
		counter:    0,
		crashAt:    MAX_DEPTH,
		schemaName: schemaName,
	}
}
