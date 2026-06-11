package contract_differ_test

import (
	"testing"

	"github.com/contracttesting/broker/internal/contract_differ"
	"github.com/contracttesting/broker/internal/model"
	"github.com/stretchr/testify/assert"
)

func newContractWithOnePetsResource(participantName string) *model.Contract {
	participant := model.NewParticipant(participantName)
	contract := model.NewContract(participant, "1", "raw")
	root := model.NewProperty("$", "object", false)
	rootId := model.NewProperty("$.id", "string", false)
	properties := map[string]model.Property{
		"$":    root,
		"$.id": rootId,
	}
	resource := model.NewProvidedRestResponse(
		"/pets",
		"get",
		"200",
		properties,
	)
	contract.AddResource(resource)
	return contract
}

func TestDiff_NoChanges_BetweenEquivalentContracts(t *testing.T) {
	prev := newContractWithOnePetsResource("pets-service")
	next := newContractWithOnePetsResource("pets-service")
	diff := contract_differ.Diff(prev, next)
	assert.Empty(t, diff.Resources)
}

func TestDiff_ReportsAddedResource(t *testing.T) {
	prev := newContractWithOnePetsResource("pets-service")
	next := newContractWithOnePetsResource("pets-service")
	root := model.NewProperty("$", "object", false)
	rootId := model.NewProperty("$.id", "string", false)
	properties := map[string]model.Property{
		"$":    root,
		"$.id": rootId,
	}
	resource := model.NewProvidedRestResponse(
		"/pets/*",
		"get",
		"200",
		properties,
	)
	next.AddResource(resource)
	diff := contract_differ.Diff(prev, next)
	assert.Len(t, diff.Resources, 1)
	for _, change := range diff.Resources {
		assert.Equal(t, contract_differ.ChangeAdded, change.Kind)
		assert.Equal(t, "/pets/*", change.Resource.Endpoint)
	}
}

func TestDiff_NextNil_AllResourcesRemoved(t *testing.T) {
	prev := newContractWithOnePetsResource("pets-service")
	diff := contract_differ.Diff(prev, nil)
	assert.Len(t, diff.Resources, 1)
	for _, change := range diff.Resources {
		assert.Equal(t, contract_differ.ChangeRemoved, change.Kind)
	}
}

func TestDiff_RemovedResource(t *testing.T) {
	oldContract := newContractWithOnePetsResource("pets-service")
	properties := map[string]model.Property{
		"$":    model.NewProperty("$", "object", false),
		"$.id": model.NewProperty("$.id", "string", false),
	}
	resource := model.NewProvidedRestResponse(
		"/pets/*",
		"get",
		"200",
		properties,
	)
	oldContract.AddResource(resource)
	newContract := newContractWithOnePetsResource("pets-service")
	diff := contract_differ.Diff(oldContract, newContract)
	assert.Len(t, diff.Resources, 1)
	for _, change := range diff.Resources {
		assert.Equal(t, contract_differ.ChangeRemoved, change.Kind)
		assert.Equal(t, "/pets/*", change.Resource.Endpoint)
		assert.Len(t, change.Properties, 2)
		for _, propChange := range change.Properties {
			assert.Equal(t, contract_differ.ChangeRemoved, propChange.Kind)
		}
	}
}

func TestDiff_ModifiedResource_PropertyAdded(t *testing.T) {
	oldContract := newContractWithOnePetsResource("pets-service")
	properties := map[string]model.Property{
		"$":      model.NewProperty("$", "object", false),
		"$.id":   model.NewProperty("$.id", "string", false),
		"$.name": model.NewProperty("$.name", "string", false),
	}
	resource := model.NewProvidedRestResponse("/pets", "get", "200", properties)
	newContract := newContractWithOnePetsResource("pets-service")
	newContract.AddResource(resource)
	diff := contract_differ.Diff(oldContract, newContract)
	assert.Len(t, diff.Resources, 1)
	for _, change := range diff.Resources {
		assert.Equal(t, contract_differ.ChangeModified, change.Kind)
		assert.Len(t, change.Properties, 1)
		assert.Equal(t, contract_differ.ChangeAdded, change.Properties["$.name"].Kind)
	}
}

func TestDiff_ModifiedResource_PropertyRemoved(t *testing.T) {
	oldContract := model.NewContract(model.NewParticipant("pets-service"), "1", "raw")
	oldContract.AddResource(
		model.NewProvidedRestResponse("/pets", "get", "200", map[string]model.Property{
			"$":      model.NewProperty("$", "object", false),
			"$.id":   model.NewProperty("$.id", "string", false),
			"$.name": model.NewProperty("$.name", "string", false),
		}),
	)
	newContract := newContractWithOnePetsResource("pets-service")
	diff := contract_differ.Diff(oldContract, newContract)
	assert.Len(t, diff.Resources, 1)
	for _, change := range diff.Resources {
		assert.Equal(t, contract_differ.ChangeModified, change.Kind)
		assert.Len(t, change.Properties, 1)
		assert.Equal(t, contract_differ.ChangeRemoved, change.Properties["$.name"].Kind)
	}
}

func TestDiff_ModifiedResource_PropertyTypeChanged(t *testing.T) {
	oldContract := newContractWithOnePetsResource("pets-service")

	newContract := model.NewContract(model.NewParticipant("pets-service"), "1", "raw")
	newContract.AddResource(model.NewProvidedRestResponse("/pets", "get", "200", map[string]model.Property{
		"$":    model.NewProperty("$", "object", false),
		"$.id": model.NewProperty("$.id", "int", false),
	}))
	diff := contract_differ.Diff(oldContract, newContract)
	assert.Len(t, diff.Resources, 1)
	for _, resourceChange := range diff.Resources {
		assert.Equal(t, contract_differ.ChangeModified, resourceChange.Kind)
		assert.Equal(t, contract_differ.ChangeModified, resourceChange.Properties["$.id"].Kind)
		assert.Equal(t, "string", resourceChange.Properties["$.id"].Before.Type)
		assert.Equal(t, "int", resourceChange.Properties["$.id"].After.Type)
	}
}
