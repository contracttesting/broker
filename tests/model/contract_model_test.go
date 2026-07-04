package model_test

import (
	"testing"

	"github.com/contracttesting/broker/internal/model"
	"github.com/stretchr/testify/assert"
)

func newContractWithOnePetsResource(participantName string) *model.UploadedContract {
	contract := model.NewUploadedContract(0, participantName, "1", "raw")
	contract.AddResource(model.NewRestResponseProvider("/pets", "get", "200", map[string]model.Property{
		"$":    model.NewProperty("$", "object", false),
		"$.id": model.NewProperty("$.id", "string", false),
	}))
	return contract
}

func TestContract_Checksum_IsStableForEquivalentContracts(t *testing.T) {
	a := newContractWithOnePetsResource("pets-service")
	b := newContractWithOnePetsResource("pets-service")

	assert.Equal(t, a.Checksum(), b.Checksum())
}

func TestContract_Checksum_DiffersWhenResourceAdded(t *testing.T) {
	a := newContractWithOnePetsResource("pets-service")

	b := newContractWithOnePetsResource("pets-service")
	b.AddResource(model.NewRestResponseProvider("/pets/*", "get", "200", map[string]model.Property{
		"$": model.NewProperty("$", "object", false),
	}))

	assert.NotEqual(t, a.Checksum(), b.Checksum())
}

// Locks in the JSON-marshal invariant: identical content added in a different
// order must yield the same checksum (encoding/json sorts map keys recursively).
func TestContract_Checksum_IsOrderIndependent(t *testing.T) {
	first := model.NewRestResponseProvider("/pets", "get", "200", map[string]model.Property{
		"$":    model.NewProperty("$", "object", false),
		"$.id": model.NewProperty("$.id", "string", false),
	})
	second := model.NewRestResponseProvider("/pets/*", "get", "200", map[string]model.Property{
		"$": model.NewProperty("$", "object", false),
	})

	a := model.NewUploadedContract(0, "pets-service", "1", "raw")
	a.AddResource(first)
	a.AddResource(second)

	b := model.NewUploadedContract(0, "pets-service", "1", "raw")
	b.AddResource(second)
	b.AddResource(first)

	assert.Equal(t, a.Checksum(), b.Checksum())
}
