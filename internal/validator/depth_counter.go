package validator

const MAX_DEPTH = 10

type DepthCounter struct {
	levels int
}

func NewDepthCounter() *DepthCounter {
	return &DepthCounter{
		levels: 0,
	}
}

func (dc *DepthCounter) Deeper() {
	dc.levels++
}

func (dc *DepthCounter) Exceeded() bool {
	return dc.levels >= MAX_DEPTH
}
