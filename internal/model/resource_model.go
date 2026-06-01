package model

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

const (
	Consumes     Direction    = "consumes"
	Provides     Direction    = "provides"
	RestRequest  ResourceKind = "rest_request"
	RestResponse ResourceKind = "rest_response"
)

type Direction string

func (direction *Direction) String() string {
	return string(*direction)
}

type ResourceKind string

func (resourceKind *ResourceKind) String() string {
	return string(*resourceKind)
}

type Resource struct {
	ID                 int64               `json:"-"`
	Direction          Direction           `json:"direction"`
	Kind               ResourceKind        `json:"kind"`
	ConsumedProvider   string              `json:"consumed_provider"`
	Endpoint           string              `json:"endpoint"`
	Method             string              `json:"method"`
	ResponseStatusCode string              `json:"response_status_code"`
	Properties         map[string]Property `json:"-"`
	Version            string              `json:"version"`
	Participant        *Participant        `json:"-"`
}

// Operation renders the HTTP operation the resource describes, with the method
// upper-cased and the message kind made explicit: "POST /accounts (request)" or
// "POST /accounts (response 201)".
func (resouce *Resource) Operation() string {
	operation := fmt.Sprintf("%s %s", strings.ToUpper(resouce.Method), resouce.Endpoint)
	if resouce.Kind == RestResponse {
		return operation + fmt.Sprintf(" (response %s)", resouce.ResponseStatusCode)
	}

	return operation + " (request)"
}

func (resouce *Resource) AddParticipant(participant *Participant) {
	resouce.Participant = participant
}

func (resouce *Resource) ParticipantID() int64 {
	return resouce.Participant.ID
}

func (resouce *Resource) ProviderHash() string {
	providerName := resouce.ConsumedProvider
	if resouce.Direction == Provides {
		providerName = resouce.ParticipantName()
	}

	parts := []string{providerName, resouce.Endpoint, resouce.Method}
	if resouce.Kind == RestResponse {
		parts = append(parts, resouce.ResponseStatusCode)
	}

	return hashParts(parts)
}

func (resouce *Resource) ConsumerHash() string {
	if resouce.Direction != Consumes {
		return ""
	}

	parts := []string{resouce.ParticipantName(), resouce.ConsumedProvider, resouce.Endpoint, resouce.Method}
	if resouce.Kind == RestResponse {
		parts = append(parts, resouce.ResponseStatusCode)
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
		string(resouce.Kind),
		resouce.ParticipantName(),
		resouce.ConsumedProvider,
		resouce.Endpoint,
		resouce.Method,
		resouce.ResponseStatusCode,
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
	return &Resource{
		Direction:        Consumes,
		Kind:             RestRequest,
		ConsumedProvider: provider,
		Endpoint:         endpoint,
		Method:           method,
		Properties:       properties,
	}
}

func NewProvidedRestRequest(
	endpoint, method string,
	properties map[string]Property,
) *Resource {
	return &Resource{
		Direction:  Provides,
		Kind:       RestRequest,
		Endpoint:   endpoint,
		Method:     method,
		Properties: properties,
	}
}

func NewConsumedRestResponse(
	provider, endpoint, method, statusCode string,
	properties map[string]Property,
) *Resource {
	return &Resource{
		Direction:          Consumes,
		Kind:               RestResponse,
		ConsumedProvider:   provider,
		Endpoint:           endpoint,
		Method:             method,
		ResponseStatusCode: statusCode,
		Properties:         properties,
	}
}

func NewProvidedRestResponse(
	endpoint, method, statusCode string,
	properties map[string]Property,
) *Resource {
	return &Resource{
		Direction:          Provides,
		Kind:               RestResponse,
		Endpoint:           endpoint,
		Method:             method,
		ResponseStatusCode: statusCode,
		Properties:         properties,
	}
}
