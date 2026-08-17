package model_test

import (
	"testing"

	"github.com/contracttesting/broker/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const petsSource = "pets.yaml"

func newContractWithOnePetsResource(participantName string) *model.UploadedContract {
	contract := model.NewUploadedContract(0, participantName, "1", "raw")
	_ = contract.AddResource(model.NewRestResponseProvider("/pets", "get", "200", map[string]model.Property{
		"$":    model.NewProperty("$", "object", false),
		"$.id": model.NewProperty("$.id", "string", false),
	}), petsSource)
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
	require.NoError(t, b.AddResource(model.NewRestResponseProvider("/pets/*", "get", "200", map[string]model.Property{
		"$": model.NewProperty("$", "object", false),
	}), petsSource))

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
	require.NoError(t, a.AddResource(first, petsSource))
	require.NoError(t, a.AddResource(second, petsSource))

	b := model.NewUploadedContract(0, "pets-service", "1", "raw")
	require.NoError(t, b.AddResource(second, petsSource))
	require.NoError(t, b.AddResource(first, petsSource))

	assert.Equal(t, a.Checksum(), b.Checksum())
}

func TestContract_AddResource_RejectsAlreadyAddedHash(t *testing.T) {
	contract := newContractWithOnePetsResource("pets-service")

	err := contract.AddResource(model.NewRestResponseProvider("/pets", "get", "200", map[string]model.Property{
		"$": model.NewProperty("$", "object", false),
	}), "store.yaml")

	require.EqualError(
		t,
		err,
		"resource already added: provides GET /pets 200 from pets.yaml and store.yaml",
	)
	assert.Len(t, contract.Resources, 1)
}

func TestContract_AddResource_DescribesEachResourceShapeInTheInvariant(t *testing.T) {
	cases := []struct {
		name     string
		resource *model.UploadedResource
		want     string
	}{
		{
			name:     "provides response",
			resource: model.NewRestResponseProvider("/pets", "get", "200", nil),
			want:     "resource already added: provides GET /pets 200 from pets.yaml and store.yaml",
		},
		{
			name:     "provides request",
			resource: model.NewRestRequestProvider("/pets", "post", nil),
			want:     "resource already added: provides POST /pets request from pets.yaml and store.yaml",
		},
		{
			name:     "consumes response",
			resource: model.NewRestResponseConsumer("payments", "/invoices", "get", "200", nil),
			want:     "resource already added: consumes payments GET /invoices 200 from pets.yaml and store.yaml",
		},
		{
			name:     "consumes request",
			resource: model.NewRestRequestConsumer("payments", "/invoices", "post", nil),
			want:     "resource already added: consumes payments POST /invoices request from pets.yaml and store.yaml",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			contract := model.NewUploadedContract(0, "pets-service", "1", "raw")
			require.NoError(t, contract.AddResource(testCase.resource, petsSource))

			require.EqualError(t, contract.AddResource(testCase.resource, "store.yaml"), testCase.want)
		})
	}
}
