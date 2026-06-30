package model_test

import (
	"testing"

	"github.com/contracttesting/broker/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestResource_ProviderHash_BridgesConsumerAndProvider(t *testing.T) {
	consumer := model.NewRestResponseConsumer("pets-service", "/pets", "get", "200", nil)
	provider := model.NewRestResponseProvider("/pets", "get", "200", nil)

	// A consumer derives the provider hash from the provider it consumes; the
	// provider derives it from its own participant name. They must match.
	assert.Equal(t, consumer.ProviderHash("web-app"), provider.ProviderHash("pets-service"))
}

func TestResource_ConsumerHash_EmptyForProvidedResource(t *testing.T) {
	provider := model.NewRestResponseProvider("/pets", "get", "200", nil)

	assert.Empty(t, provider.ConsumerHash("pets-service"))
}

func TestResource_PrimaryHash_ConsumerDirection_EqualsConsumerHash(t *testing.T) {
	consumer := model.NewRestResponseConsumer("pets-service", "/pets", "get", "200", nil)

	assert.Equal(t, consumer.ConsumerHash("web-app"), consumer.PrimaryHash("web-app"))
}

func TestResource_ConsumerHash_RestResponse_IncludesStatusCode(t *testing.T) {
	a := model.NewRestResponseConsumer("pets-service", "/pets", "get", "200", nil)
	b := model.NewRestResponseConsumer("pets-service", "/pets", "get", "404", nil)

	assert.NotEqual(t, a.ConsumerHash("web-app"), b.ConsumerHash("web-app"))
}
