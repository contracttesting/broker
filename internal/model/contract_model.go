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

// AddResource keys the resource by its hash and rejects a hash already taken.
func (contract *UploadedContract) AddResource(resource *UploadedResource) error {
	if contract.Resources == nil {
		contract.Resources = make(map[string]UploadedResource)
	}

	resource.ParticipantName = contract.ParticipantName
	hash := resource.PrimaryHash()

	if _, taken := contract.Resources[hash]; taken {
		return fmt.Errorf("resource already added: %s", resource.Describe())
	}

	contract.Resources[hash] = *resource

	return nil
}

func (contract *UploadedContract) Checksum() string {
	payload, _ := json.Marshal(contract.Resources)
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
