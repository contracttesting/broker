package model

import (
	"strings"

	"github.com/guregu/null"
)

type UploadedResource struct {
	ParticipantName    string
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

// Describe names the resource the way publish errors quote it: the same parts that
// make up its hash, in reading order.
func (r *UploadedResource) Describe() string {
	parts := []string{r.Direction.String()}

	if r.IsConsumer() && r.ConsumedProvider.String != "" {
		parts = append(parts, r.ConsumedProvider.String)
	}

	parts = append(parts, strings.ToUpper(r.Method), r.Endpoint)

	if r.Interaction == RestResponse {
		parts = append(parts, r.ResponseStatusCode.String)
	} else {
		parts = append(parts, "request")
	}

	return strings.Join(parts, " ")
}

func (r *UploadedResource) ProviderHash() string {
	providerName := r.ParticipantName
	if r.IsConsumer() {
		providerName = r.ConsumedProvider.String
	}

	parts := []string{providerName, r.Endpoint, r.Method}
	if r.Interaction == RestResponse {
		parts = append(parts, r.ResponseStatusCode.String)
	}

	return Hash(parts...)
}

func (r *UploadedResource) ConsumerHash() string {
	if !r.IsConsumer() {
		return ""
	}

	parts := []string{r.ParticipantName, r.ConsumedProvider.String, r.Endpoint, r.Method}
	if r.Interaction == RestResponse {
		parts = append(parts, r.ResponseStatusCode.String)
	}

	return Hash(parts...)
}

func (r *UploadedResource) PrimaryHash() string {
	if r.IsProvider() {
		return r.ProviderHash()
	}

	return r.ConsumerHash()
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
