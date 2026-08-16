package repository

import (
	"database/sql"
	"time"

	"github.com/contracttesting/broker/internal/contract_differ"
	"github.com/contracttesting/broker/internal/model"
	"github.com/guregu/null"
)

// This struct is used to build the entire contract avoiding N+1 problem
type tableRow struct {
	// Participant
	ParticipantID   int64
	ParticipantName string

	// Contract
	ContractID        int64
	ContractVersion   string
	ContractContent   string
	ContractCreatedAt time.Time

	// Resource
	ResourceID                 int64
	ResourceDirection          string
	ResourceInteraction        string
	ResourceConsumedProvider   sql.NullString
	ResourceEndpoint           string
	ResourceMethod             string
	ResourceResponseStatusCode sql.NullString
	ResourceProviderHash       string
	ResourceConsumerHash       sql.NullString
	ResourceCreatedAt          time.Time
	ResourceVersion            string
	ResourceVersionChangeType  string

	// Property
	PropertyID                int64
	PropertyPath              string
	PropertyVersionType       sql.NullString
	PropertyVersionOptional   sql.NullBool
	PropertyVersionChangeType string

	// Deploy & Environment
	DeploymentEnvironment sql.NullString
	DeploymentVersion     sql.NullString
}

func (c *tableRow) toPersistedContractModel() *model.PersistedContract {
	return &model.PersistedContract{
		ID:              c.ContractID,
		ParticipantID:   c.ParticipantID,
		ParticipantName: c.ParticipantName,
		Version:         c.ContractVersion,
		ContractContent: c.ContractContent,
		Resources:       make(map[string]model.PersistedResource),
	}
}

func (c *tableRow) toResourceModel() model.PersistedResource {
	resource := model.PersistedResource{
		Direction:        model.Direction(c.ResourceDirection),
		Interaction:      model.Interaction(c.ResourceInteraction),
		Endpoint:         c.ResourceEndpoint,
		Method:           c.ResourceMethod,
		Properties:       make(map[string]model.Property),
		DeployedVersions: make(map[string]string),
		ParticipantName:  c.ParticipantName,
		ParticipantID:    c.ParticipantID,
		ContractID:       c.ContractID,
		ProviderHash:     c.ResourceProviderHash,
		Removed:          c.ResourceVersionChangeType == string(contract_differ.ChangeRemoved),
	}

	if c.ResourceVersion != "" {
		resource.ParticipantVersion = null.StringFrom(c.ResourceVersion)
	}

	if c.ResourceConsumedProvider.String != "" {
		resource.ConsumedProvider = null.StringFrom(c.ResourceConsumedProvider.String)
	}

	if c.ResourceResponseStatusCode.String != "" {
		resource.ResponseStatusCode = null.StringFrom(c.ResourceResponseStatusCode.String)
	}

	if c.ResourceConsumerHash.String != "" {
		resource.ConsumerHash = null.StringFrom(c.ResourceConsumerHash.String)
	}

	return resource
}

func (c *tableRow) toPropertyModel() model.Property {
	return model.Property{
		ID:       c.PropertyID,
		Path:     c.PropertyPath,
		Type:     c.PropertyVersionType.String,
		Optional: c.PropertyVersionOptional.Bool,
	}
}

type insertPropertyVersionRow struct {
	PropertyID int64
	ContractID int64
	Type       sql.NullString
	Optional   sql.NullBool
	ChangeType string
}

func newInsertPropertyVersionRowAdded(contractID, propertyID int64, p model.Property) *insertPropertyVersionRow {
	return newInsertPropertyVersionRow(contractID, propertyID, p, contract_differ.ChangeAdded)
}

func newInsertPropertyVersionRowModified(contractID, propertyID int64, p model.Property) *insertPropertyVersionRow {
	return newInsertPropertyVersionRow(contractID, propertyID, p, contract_differ.ChangeModified)
}

func newInsertPropertyVersionRowRemoved(contractID, propertyID int64, p model.Property) *insertPropertyVersionRow {
	return newInsertPropertyVersionRow(contractID, propertyID, p, contract_differ.ChangeRemoved)
}

func newInsertPropertyVersionRow(contractID, propertyID int64, p model.Property, change contract_differ.ChangeKind) *insertPropertyVersionRow {
	return &insertPropertyVersionRow{
		PropertyID: propertyID,
		ContractID: contractID,
		Type:       sql.NullString{String: p.Type, Valid: p.Type != ""},
		Optional:   sql.NullBool{Bool: p.Optional, Valid: true},
		ChangeType: string(change),
	}
}

type insertResourceVersionRow struct {
	ResourceID int64
	ContractID int64
	ChangeType string
}

func newInsertResourceVersionRowAdded(contractID, resourceID int64) *insertResourceVersionRow {
	return &insertResourceVersionRow{
		ResourceID: resourceID,
		ContractID: contractID,
		ChangeType: string(contract_differ.ChangeAdded),
	}
}

func newInsertResourceVersionRowRemoved(contractID, resourceID int64) *insertResourceVersionRow {
	return &insertResourceVersionRow{
		ResourceID: resourceID,
		ContractID: contractID,
		ChangeType: string(contract_differ.ChangeRemoved),
	}
}
