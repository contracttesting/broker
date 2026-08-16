package dsl

import "fmt"

const MAX_DEPTH = 10

// SchemaTooDeep is the error returned when a schema nests past MAX_DEPTH, which is
// also how a cyclic chain of refs terminates. Hydration propagates it up as a publish
// error.
type SchemaTooDeep struct {
	SchemaName string
	MaxDepth   int
}

func (e SchemaTooDeep) Error() string {
	return fmt.Sprintf(
		"schema %s is too deep with more than %d levels",
		e.SchemaName,
		e.MaxDepth,
	)
}

type DepthCounter struct {
	counter    int
	limit      int
	schemaName string
}

func (dc *DepthCounter) Enter() error {
	dc.counter = dc.counter + 1

	if dc.counter >= dc.limit {
		return SchemaTooDeep{SchemaName: dc.schemaName, MaxDepth: dc.limit}
	}

	return nil
}

func (dc *DepthCounter) Exit() {
	dc.counter = dc.counter - 1
}

func NewDepthCounter(schemaName string) *DepthCounter {
	return &DepthCounter{
		counter:    0,
		limit:      MAX_DEPTH,
		schemaName: schemaName,
	}
}
