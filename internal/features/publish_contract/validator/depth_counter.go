package validator

import "github.com/contracttesting/broker/internal/mapper/schemamapper"

// DepthCounter is the budget one schema branch carries. It travels by value: every
// level gets a copy, so a deep branch never charges the branches beside it.
type DepthCounter struct {
	levels int
}

func (dc DepthCounter) Deeper() DepthCounter {
	return DepthCounter{levels: dc.levels + 1}
}

func (dc DepthCounter) Exceeded() bool {
	return dc.levels >= schemamapper.MaxDepth
}
