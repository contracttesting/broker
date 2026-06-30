package model

import (
	"sort"

	"github.com/guregu/null"
)

// CounterpartResource is a resource loaded from the database. It carries no hash
// (the stored hash is used as the map key at the repository boundary) and no ID
// (the publish-diff re-derives ids by natural-key lookup). Participant data is
// flat because a counterpart's participant varies per resource.
type CounterpartResource struct {
	Direction          Direction           `json:"direction"`
	Interaction        Interaction         `json:"interaction"`
	ConsumedProvider   null.String         `json:"consumed_provider"`
	Endpoint           string              `json:"endpoint"`
	Method             string              `json:"method"`
	ResponseStatusCode null.String         `json:"response_status_code"`
	Properties         map[string]Property `json:"-"`
	ParticipantName    string              `json:"-"`
	ParticipantID      int64               `json:"-"`
	ParticipantVersion null.String         `json:"version"`
	DeployedVersions   map[string]string   `json:"-"`
}

func (r *CounterpartResource) IsConsumer() bool {
	return r.Direction == Consumes
}

func (r *CounterpartResource) IsProvider() bool {
	return r.Direction == Provides
}

func (r *CounterpartResource) DeployedVersionIn(environment string) (string, bool) {
	version, ok := r.DeployedVersions[environment]
	return version, ok
}

func (r *CounterpartResource) DeployedEnvironments() []string {
	environments := make([]string, 0, len(r.DeployedVersions))

	for environment := range r.DeployedVersions {
		environments = append(environments, environment)
	}

	sort.Strings(environments)

	return environments
}
