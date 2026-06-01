package dsl_test

import (
	"testing"

	"github.com/contracttesting/broker/internal/dsl"
	"github.com/contracttesting/broker/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToResource_ConsumerRestRequest_Parses(t *testing.T) {
	path := dsl.NewResourcePath("consumes;pets-service;rest;/pets;post;request")

	resource := path.ToResource(nil)

	require.NotNil(t, resource)
	assert.Equal(t, model.Consumes, resource.Direction)
	assert.Equal(t, model.RestRequest, resource.Kind)
	assert.Equal(t, "pets-service", resource.ConsumedProvider)
	assert.Equal(t, "/pets", resource.Endpoint)
	assert.Equal(t, "post", resource.Method)
	assert.Empty(t, resource.ResponseStatusCode)
}

func TestToResource_ProviderRestRequest_Parses(t *testing.T) {
	path := dsl.NewResourcePath("provides;rest;/pets;post;request")

	resource := path.ToResource(nil)

	require.NotNil(t, resource)
	assert.Equal(t, model.Provides, resource.Direction)
	assert.Equal(t, model.RestRequest, resource.Kind)
	assert.Empty(t, resource.ConsumedProvider)
	assert.Equal(t, "/pets", resource.Endpoint)
	assert.Equal(t, "post", resource.Method)
}

func TestToResource_ProviderRestResponse_Parses(t *testing.T) {
	path := dsl.NewResourcePath("provides;rest;/pets;get;responses;200")

	resource := path.ToResource(nil)

	require.NotNil(t, resource)
	assert.Equal(t, model.Provides, resource.Direction)
	assert.Equal(t, model.RestResponse, resource.Kind)
	assert.Equal(t, "/pets", resource.Endpoint)
	assert.Equal(t, "get", resource.Method)
	assert.Equal(t, "200", resource.ResponseStatusCode)
}

func TestToResource_UnrecognizedPath_Panics(t *testing.T) {
	path := dsl.NewResourcePath("garbage;not;a;real;path")

	assert.Panics(t, func() { path.ToResource(nil) })
}
