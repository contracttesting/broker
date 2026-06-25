package model

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/guregu/null"
)

const (
	Consumes     Direction   = "consumes"
	Provides     Direction   = "provides"
	RestRequest  Interaction = "rest_request"
	RestResponse Interaction = "rest_response"
)

type Direction string

func (direction *Direction) String() string {
	return string(*direction)
}

type Interaction string

func (interaction *Interaction) String() string {
	return string(*interaction)
}

type Resource struct {
	ID                 int64               `json:"-"`
	Direction          Direction           `json:"direction"`
	Interaction        Interaction         `json:"interaction"`
	ConsumedProvider   null.String         `json:"consumed_provider"`
	Endpoint           string              `json:"endpoint"`
	Method             string              `json:"method"`
	ResponseStatusCode null.String         `json:"response_status_code"`
	Properties         map[string]Property `json:"-"`
	DeployedVersions   map[string]string   `json:"-"`
	Version            null.String         `json:"version"`
	Participant        *Participant        `json:"-"`
}

func (resouce *Resource) Operation() string {
	operation := fmt.Sprintf("%s %s", strings.ToUpper(resouce.Method), resouce.Endpoint)
	if resouce.Interaction == RestResponse {
		return operation + fmt.Sprintf(" (response %s)", resouce.ResponseStatusCode)
	}

	return operation + " (request)"
}

func (resouce *Resource) DeployedEnvironments() []string {
	environments := make([]string, 0, len(resouce.DeployedVersions))
	for environment := range resouce.DeployedVersions {
		environments = append(environments, environment)
	}
	sort.Strings(environments)

	return environments
}

func (resouce *Resource) DeployedVersionIn(environment string) (string, bool) {
	version, ok := resouce.DeployedVersions[environment]
	return version, ok
}

func (resouce *Resource) AddParticipant(participant *Participant) {
	resouce.Participant = participant
}

func (resouce *Resource) ParticipantID() int64 {
	return resouce.Participant.ID
}

func (resouce *Resource) IsConsumer() bool {
	return resouce.Direction == Consumes
}

func (resouce *Resource) IsProvider() bool {
	return resouce.Direction == Provides
}

func (resouce *Resource) ProviderHash() string {
	providerName := resouce.ConsumedProvider.String
	if resouce.IsProvider() {
		providerName = resouce.ParticipantName()
	}

	parts := []string{providerName, resouce.Endpoint, resouce.Method}
	if resouce.Interaction == RestResponse {
		parts = append(parts, resouce.ResponseStatusCode.String)
	}

	return hashParts(parts)
}

func (resouce *Resource) ConsumerHash() string {
	if resouce.Direction != Consumes {
		return ""
	}

	parts := []string{resouce.ParticipantName(), resouce.ConsumedProvider.String, resouce.Endpoint, resouce.Method}
	if resouce.Interaction == RestResponse {
		parts = append(parts, resouce.ResponseStatusCode.String)
	}

	return hashParts(parts)
}

func (resouce *Resource) ParticipantName() string {
	return resouce.Participant.Name
}

func (resouce *Resource) CanonicalKey() string {
	propertyKeys := make([]string, 0, len(resouce.Properties))

	for _, property := range resouce.Properties {
		propertyKeys = append(propertyKeys, property.CanonicalKey())
	}

	sort.Strings(propertyKeys)

	return strings.Join([]string{
		string(resouce.Direction),
		string(resouce.Interaction),
		resouce.ParticipantName(),
		resouce.ConsumedProvider.String,
		resouce.Endpoint,
		resouce.Method,
		resouce.ResponseStatusCode.String,
		strings.Join(propertyKeys, ";;"),
	}, ";;")
}

func hashParts(parts []string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, ";;")))
	return hex.EncodeToString(sum[:])
}

func NewConsumedRestRequest(
	provider, endpoint, method string,
	properties map[string]Property,
) *Resource {
	resource := &Resource{
		Direction:   Consumes,
		Interaction: RestRequest,
		Endpoint:    endpoint,
		Method:      method,
		Properties:  properties,
	}

	if provider != "" {
		resource.ConsumedProvider = null.StringFrom(provider)
	}

	return resource
}

func NewProvidedRestRequest(
	endpoint, method string,
	properties map[string]Property,
) *Resource {
	return &Resource{
		Direction:   Provides,
		Interaction: RestRequest,
		Endpoint:    endpoint,
		Method:      method,
		Properties:  properties,
	}
}

func NewConsumedRestResponse(
	provider, endpoint, method, statusCode string,
	properties map[string]Property,
) *Resource {
	resource := &Resource{
		Direction:   Consumes,
		Interaction: RestResponse,
		Endpoint:    endpoint,
		Method:      method,
		Properties:  properties,
	}

	if provider != "" {
		resource.ConsumedProvider = null.StringFrom(provider)
	}

	if statusCode != "" {
		resource.ResponseStatusCode = null.StringFrom(statusCode)
	}

	return resource
}

func NewProvidedRestResponse(
	endpoint, method, statusCode string,
	properties map[string]Property,
) *Resource {
	resource := &Resource{
		Direction:   Provides,
		Interaction: RestResponse,
		Endpoint:    endpoint,
		Method:      method,
		Properties:  properties,
	}

	if statusCode != "" {
		resource.ResponseStatusCode = null.StringFrom(statusCode)
	}

	return resource
}
