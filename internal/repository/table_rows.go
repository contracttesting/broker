package repository

import (
	"database/sql"
	"time"

	"github.com/contracttesting/broker/internal/contract_differ"
	"github.com/contracttesting/broker/internal/model"
	"github.com/guregu/null"
)

type tableRow struct {
	ParticipantID   int64
	ParticipantName string

	ContractID          int64
	ContractVersion     string
	ContractRawContract string
	ContractCreatedAt   time.Time

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

	PropertyID   int64
	PropertyPath string

	PropertyVersionType       sql.NullString
	PropertyVersionOptional   sql.NullBool
	PropertyVersionChangeType string

	DeploymentEnvironment sql.NullString
	DeploymentVersion     sql.NullString
}

func (c *tableRow) toContractModel() *model.Contract {
	return &model.Contract{
		ID:          c.ContractID,
		Version:     c.ContractVersion,
		RawContract: c.ContractRawContract,
		Resources:   make(map[string]model.CounterpartResource),
		Participant: &model.Participant{
			ID:   c.ParticipantID,
			Name: c.ParticipantName,
		},
	}
}

func (c *tableRow) toResourceModel() model.CounterpartResource {
	resource := model.CounterpartResource{
		Direction:        model.Direction(c.ResourceDirection),
		Interaction:      model.Interaction(c.ResourceInteraction),
		Endpoint:         c.ResourceEndpoint,
		Method:           c.ResourceMethod,
		Properties:       make(map[string]model.Property),
		DeployedVersions: make(map[string]string),
		ParticipantName:  c.ParticipantName,
		ParticipantID:    c.ParticipantID,
	}

	if c.ResourceConsumedProvider.String != "" {
		resource.ConsumedProvider = null.StringFrom(c.ResourceConsumedProvider.String)
	}

	if c.ResourceResponseStatusCode.String != "" {
		resource.ResponseStatusCode = null.StringFrom(c.ResourceResponseStatusCode.String)
	}

	if c.ResourceVersion != "" {
		resource.ParticipantVersion = null.StringFrom(c.ResourceVersion)
	}

	return resource
}

// primaryHash returns the resource's own identity hash as stored in the row —
// provider_hash for providers, consumer_hash for consumers. It is the loaded
// contract's map key (no recompute).
func (c *tableRow) primaryHash() string {
	if c.ResourceDirection == string(model.Provides) {
		return c.ResourceProviderHash
	}
	return c.ResourceConsumerHash.String
}

func (c *tableRow) toPropertyModel() model.Property {
	return model.Property{
		ID:       c.PropertyID,
		Path:     c.PropertyPath,
		Type:     c.PropertyVersionType.String,
		Optional: c.PropertyVersionOptional.Bool,
	}
}

func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
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
		Type:       nullString(p.Type),
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
