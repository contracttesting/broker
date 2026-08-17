package dsl

const MAX_DEPTH = 10

// DepthCounter is the budget a schema descent carries in its Position. It is copied at
// every level, so a deep branch never charges the branches beside it, and a cyclic
// chain of refs terminates once the budget runs out.
type DepthCounter struct {
	levels int
}

func (dc DepthCounter) Deeper() DepthCounter {
	return DepthCounter{levels: dc.levels + 1}
}

func (dc DepthCounter) Exceeded() bool {
	return dc.levels >= MAX_DEPTH
}
