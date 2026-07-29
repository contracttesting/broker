package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

type UploadedContract struct {
	ID              int64
	Version         string
	RawContract     string
	Resources       map[string]UploadedResource
	ParticipantID   int64
	ParticipantName string
}

func NewUploadedContract(
	participantID int64,
	participantName string,
	version string,
	rawContract string,
) *UploadedContract {
	return &UploadedContract{
		ParticipantID:   participantID,
		ParticipantName: participantName,
		Version:         version,
		RawContract:     rawContract,
	}
}

func (contract *UploadedContract) AddResource(resource *UploadedResource) {
	if contract.Resources == nil {
		contract.Resources = make(map[string]UploadedResource)
	}

	resource.ParticipantName = contract.ParticipantName
	contract.Resources[resource.PrimaryHash()] = *resource
}

func (contract *UploadedContract) Checksum() string {
	payload, _ := json.Marshal(contract.Resources)
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
