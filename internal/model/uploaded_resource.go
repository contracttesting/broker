package model

import (
	"sort"
	"strings"

	"github.com/guregu/null"
)

type UploadedResource struct {
	Direction          Direction
	Interaction        Interaction
	ConsumedProvider   null.String
	Endpoint           string
	Method             string
	ResponseStatusCode null.String
	Properties         map[string]Property
}

func (r *UploadedResource) IsConsumer() bool {
	return r.Direction == Consumes
}

func (r *UploadedResource) IsProvider() bool {
	return r.Direction == Provides
}

func (r *UploadedResource) ProviderHash(participant string) string {
	providerName := participant
	if r.IsConsumer() {
		providerName = r.ConsumedProvider.String
	}

	parts := []string{providerName, r.Endpoint, r.Method}
	if r.Interaction == RestResponse {
		parts = append(parts, r.ResponseStatusCode.String)
	}

	return Hash(parts...)
}

func (r *UploadedResource) ConsumerHash(participant string) string {
	if !r.IsConsumer() {
		return ""
	}

	parts := []string{participant, r.ConsumedProvider.String, r.Endpoint, r.Method}
	if r.Interaction == RestResponse {
		parts = append(parts, r.ResponseStatusCode.String)
	}

	return Hash(parts...)
}

func (r *UploadedResource) PrimaryHash(participant string) string {
	if r.IsProvider() {
		return r.ProviderHash(participant)
	}

	return r.ConsumerHash(participant)
}

func (r *UploadedResource) CanonicalKey(participant string) string {
	propertyKeys := make([]string, 0, len(r.Properties))

	for _, property := range r.Properties {
		propertyKeys = append(propertyKeys, property.CanonicalKey())
	}

	sort.Strings(propertyKeys)

	return strings.Join([]string{
		string(r.Direction),
		string(r.Interaction),
		participant,
		r.ConsumedProvider.String,
		r.Endpoint,
		r.Method,
		r.ResponseStatusCode.String,
		strings.Join(propertyKeys, ";;"),
	}, ";;")
}

func NewRestRequestConsumer(
	provider, endpoint, method string,
	properties map[string]Property,
) *UploadedResource {
	resource := &UploadedResource{
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

func NewRestRequestProvider(
	endpoint, method string,
	properties map[string]Property,
) *UploadedResource {
	return &UploadedResource{
		Direction:   Provides,
		Interaction: RestRequest,
		Endpoint:    endpoint,
		Method:      method,
		Properties:  properties,
	}
}

func NewRestResponseConsumer(
	provider, endpoint, method, statusCode string,
	properties map[string]Property,
) *UploadedResource {
	resource := &UploadedResource{
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

func NewRestResponseProvider(
	endpoint, method, statusCode string,
	properties map[string]Property,
) *UploadedResource {
	resource := &UploadedResource{
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
