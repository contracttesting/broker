package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

type UploadedContract struct {
	ID              int64
	Version         string
	ContractContent string
	Resources       map[string]UploadedResource
	ParticipantID   int64
	ParticipantName string

	resourceSources map[string]string
}

func NewUploadedContract(
	participantID int64,
	participantName string,
	version string,
	contractContent string,
) *UploadedContract {
	return &UploadedContract{
		ParticipantID:   participantID,
		ParticipantName: participantName,
		Version:         version,
		ContractContent: contractContent,
	}
}

// AddResource rejects a resource whose hash is already taken: two resources with the
// same hash are the same resource. Publish validation rejects a redeclaration before
// the build starts, so a hit here is a broken invariant rather than bad input.
func (contract *UploadedContract) AddResource(resource *UploadedResource, source string) error {
	if contract.Resources == nil {
		contract.Resources = make(map[string]UploadedResource)
		contract.resourceSources = make(map[string]string)
	}

	resource.ParticipantName = contract.ParticipantName
	hash := resource.PrimaryHash()

	if declaredIn, taken := contract.resourceSources[hash]; taken {
		return fmt.Errorf(
			"resource already added: %s from %s and %s",
			resource.Describe(),
			declaredIn,
			source,
		)
	}

	contract.Resources[hash] = *resource
	contract.resourceSources[hash] = source

	return nil
}

func (contract *UploadedContract) Checksum() string {
	payload, _ := json.Marshal(contract.Resources)
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
