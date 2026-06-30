package model

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// UploadedContract is built by hydrating a published contract. It owns the
// participant and supplies the participant name wherever a resource hash is
// needed (the in-memory map key and the checksum).
type UploadedContract struct {
	ID          int64
	Version     string
	RawContract string
	Resources   map[string]UploadedResource
	Participant *Participant
}

func NewUploadedContract(participant *Participant, version string, rawContract string) *UploadedContract {
	return &UploadedContract{
		Participant: participant,
		Version:     version,
		RawContract: rawContract,
	}
}

func (contract *UploadedContract) ParticipantID() int64 {
	return contract.Participant.ID
}

func (contract *UploadedContract) AddResource(resource *UploadedResource) {
	if contract.Resources == nil {
		contract.Resources = make(map[string]UploadedResource)
	}

	contract.Resources[resource.PrimaryHash(contract.Participant.Name)] = *resource
}

func (contract *UploadedContract) CanonicalKey() string {
	resourceKeys := make([]string, 0, len(contract.Resources))

	for _, resource := range contract.Resources {
		resourceKeys = append(resourceKeys, resource.CanonicalKey(contract.Participant.Name))
	}

	sort.Strings(resourceKeys)

	return strings.Join([]string{
		contract.Participant.Name,
		strings.Join(resourceKeys, ";;"),
	}, ";;")
}

func (contract *UploadedContract) Checksum() string {
	sum := sha256.Sum256([]byte(contract.CanonicalKey()))
	return hex.EncodeToString(sum[:])
}

// Contract is a contract loaded from the database. Its resources are
// CounterpartResource values keyed by their stored identity hash.
type Contract struct {
	ID          int64
	Version     string
	RawContract string
	Resources   map[string]CounterpartResource
	Participant *Participant
}

func (contract *Contract) ParticipantID() int64 {
	return contract.Participant.ID
}
