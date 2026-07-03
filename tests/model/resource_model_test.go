package model_test

import (
	"testing"

	"github.com/contracttesting/broker/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestResource_ProviderHash_BridgesConsumerAndProvider(t *testing.T) {
	consumer := model.NewRestResponseConsumer("pets-service", "/pets", "get", "200", nil)
	consumer.ParticipantName = "web-app"
	provider := model.NewRestResponseProvider("/pets", "get", "200", nil)
	provider.ParticipantName = "pets-service"

	// A consumer derives the provider hash from the provider it consumes; the
	// provider derives it from its own participant name. They must match.
	assert.Equal(t, consumer.ProviderHash(), provider.ProviderHash())
}

func TestResource_ConsumerHash_EmptyForProvidedResource(t *testing.T) {
	provider := model.NewRestResponseProvider("/pets", "get", "200", nil)
	provider.ParticipantName = "pets-service"

	assert.Empty(t, provider.ConsumerHash())
}

func TestResource_PrimaryHash_ConsumerDirection_EqualsConsumerHash(t *testing.T) {
	consumer := model.NewRestResponseConsumer("pets-service", "/pets", "get", "200", nil)
	consumer.ParticipantName = "web-app"

	assert.Equal(t, consumer.ConsumerHash(), consumer.PrimaryHash())
}

func TestResource_ConsumerHash_RestResponse_IncludesStatusCode(t *testing.T) {
	a := model.NewRestResponseConsumer("pets-service", "/pets", "get", "200", nil)
	a.ParticipantName = "web-app"
	b := model.NewRestResponseConsumer("pets-service", "/pets", "get", "404", nil)
	b.ParticipantName = "web-app"

	assert.NotEqual(t, a.ConsumerHash(), b.ConsumerHash())
}
