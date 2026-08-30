package model

type ChangeKind string

const (
	ChangeAdded    ChangeKind = "added"
	ChangeModified ChangeKind = "modified"
	ChangeRemoved  ChangeKind = "removed"
)

type ContractDiff struct {
	Resources map[string]ResourceChange
}

type ResourceChange struct {
	Kind       ChangeKind
	Properties map[string]PropertyChange
}

type PropertyChange struct {
	Kind   ChangeKind
	Before Property
	After  Property
}

type ResourceProperties = map[string]Property
