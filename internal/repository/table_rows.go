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
	ResourceKind               string
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
		Resources:   make(map[string]model.Resource),
		Participant: &model.Participant{
			ID:   c.ParticipantID,
			Name: c.ParticipantName,
		},
	}
}

func (c *tableRow) toResourceModel() model.Resource {
	resource := model.Resource{
		ID:               c.ResourceID,
		Direction:        model.Direction(c.ResourceDirection),
		Kind:             model.ResourceKind(c.ResourceKind),
		Endpoint:         c.ResourceEndpoint,
		Method:           c.ResourceMethod,
		Properties:       make(map[string]model.Property),
		DeployedVersions: make(map[string]string),
		Participant: &model.Participant{
			ID:   c.ParticipantID,
			Name: c.ParticipantName,
		},
	}

	if c.ResourceConsumedProvider.String != "" {
		resource.ConsumedProvider = null.StringFrom(c.ResourceConsumedProvider.String)
	}

	if c.ResourceResponseStatusCode.String != "" {
		resource.ResponseStatusCode = null.StringFrom(c.ResourceResponseStatusCode.String)
	}

	if c.ResourceVersion != "" {
		resource.Version = null.StringFrom(c.ResourceVersion)
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

func newInsertPropertyVersionRowAdded(c *model.Contract, p model.Property) *insertPropertyVersionRow {
	return newInsertPropertyVersionRow(c, p, contract_differ.ChangeAdded)
}

func newInsertPropertyVersionRowModified(c *model.Contract, p model.Property) *insertPropertyVersionRow {
	return newInsertPropertyVersionRow(c, p, contract_differ.ChangeModified)
}

func newInsertPropertyVersionRowRemoved(c *model.Contract, p model.Property) *insertPropertyVersionRow {
	return newInsertPropertyVersionRow(c, p, contract_differ.ChangeRemoved)
}

func newInsertPropertyVersionRow(c *model.Contract, p model.Property, change contract_differ.ChangeKind) *insertPropertyVersionRow {
	return &insertPropertyVersionRow{
		PropertyID: p.ID,
		ContractID: c.ID,
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

func newInsertResourceVersionRowAdded(c *model.Contract, r model.Resource) *insertResourceVersionRow {
	return &insertResourceVersionRow{
		ResourceID: r.ID,
		ContractID: c.ID,
		ChangeType: string(contract_differ.ChangeAdded),
	}
}

func newInsertResourceVersionRowRemoved(c *model.Contract, r model.Resource) *insertResourceVersionRow {
	return &insertResourceVersionRow{
		ResourceID: r.ID,
		ContractID: c.ID,
		ChangeType: string(contract_differ.ChangeRemoved),
	}
}
