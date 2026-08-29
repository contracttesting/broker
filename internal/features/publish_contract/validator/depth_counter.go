package validator

import "github.com/contracttesting/broker/internal/mapper/schemamapper"

// DepthCounter is the depth budget one schema branch carries.
type DepthCounter struct {
	levels int
}

func (dc DepthCounter) Deeper() DepthCounter {
	return DepthCounter{levels: dc.levels + 1}
}

func (dc DepthCounter) Exceeded() bool {
	return dc.levels >= schemamapper.MaxDepth
}
