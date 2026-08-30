package model

// OrderContractPair is the single canonicalization of a snapshot pair: the lower id first.
func OrderContractPair(a, b int64) (int64, int64) {
	if a <= b {
		return a, b
	}

	return b, a
}
