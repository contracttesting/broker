package model

import (
	"sort"

	"github.com/guregu/null"
)

type PersistedResource struct {
	ParticipantID      int64               `json:"-"`
	ParticipantName    string              `json:"-"`
	ContractID         int64               `json:"-"`
	Direction          Direction           `json:"direction"`
	Interaction        Interaction         `json:"interaction"`
	ConsumedProvider   null.String         `json:"consumedProvider"`
	ConsumerHash       null.String         `json:"-"`
	ProviderHash       string              `json:"-"`
	Endpoint           string              `json:"endpoint"`
	Method             string              `json:"method"`
	ResponseStatusCode null.String         `json:"responseStatusCode"`
	Properties         map[string]Property `json:"-"`
	ParticipantVersion null.String         `json:"version"`
	DeployedVersions   map[string]string   `json:"-"`
	Removed            bool                `json:"-"`
}

func (r *PersistedResource) IsConsumer() bool {
	return r.Direction == Consumes
}

func (r *PersistedResource) IsProvider() bool {
	return r.Direction == Provides
}

func (r *PersistedResource) DeployedVersionIn(environment string) (string, bool) {
	version, ok := r.DeployedVersions[environment]
	return version, ok
}

func (r *PersistedResource) PrimaryHash() string {
	if r.Direction == Provides {
		return r.ProviderHash
	}

	return r.ConsumerHash.String
}

func (r *PersistedResource) DeployedEnvironments() []string {
	environments := make([]string, 0, len(r.DeployedVersions))

	for environment := range r.DeployedVersions {
		environments = append(environments, environment)
	}

	sort.Strings(environments)

	return environments
}
