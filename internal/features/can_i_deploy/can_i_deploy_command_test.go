package can_i_deploy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConsumerResourcePrefixResolvesByDirection(t *testing.T) {
	status := "200"
	consumer := &BreakResource{Direction: "consumes", Interaction: "rest_response", Endpoint: "/payments/*", Method: "get", ResponseStatusCode: &status}
	provider := &BreakResource{Direction: "provides", Interaction: "rest_response", Endpoint: "/other/*", Method: "post", ResponseStatusCode: nil}

	t.Run("checked side is the consumer", func(t *testing.T) {
		prefix := consumerResourcePrefix(ContractBreak{CheckedResource: consumer, CounterpartResource: provider})

		assert.Equal(t, "GET /payments/* response 200", prefix)
	})

	t.Run("counterpart side is the consumer", func(t *testing.T) {
		prefix := consumerResourcePrefix(ContractBreak{CheckedResource: provider, CounterpartResource: consumer})

		assert.Equal(t, "GET /payments/* response 200", prefix)
	})

	t.Run("request interaction has no status suffix", func(t *testing.T) {
		request := &BreakResource{Direction: "consumes", Interaction: "rest_request", Endpoint: "/payments", Method: "post", ResponseStatusCode: nil}

		prefix := consumerResourcePrefix(ContractBreak{CheckedResource: request})

		assert.Equal(t, "POST /payments request", prefix)
	})
}

func TestFormatBreakLine(t *testing.T) {
	status := "200"
	consumer := &BreakResource{Direction: "consumes", Interaction: "rest_response", Endpoint: "/payments/*", Method: "GET", ResponseStatusCode: &status}
	provider := &BreakResource{Direction: "provides", Interaction: "rest_response", Endpoint: "/payments/*", Method: "GET", ResponseStatusCode: &status}

	t.Run("provider_resource_not_found", func(t *testing.T) {
		line := formatBreakLine("production", ContractBreak{
			CheckedResource: consumer,
			Reason:          "provider_resource_not_found",
		})

		assert.Equal(t, `GET /payments/* response 200: no matching resource in provider`, line)
	})

	t.Run("provider_resource_not_deployed_in_environment", func(t *testing.T) {
		line := formatBreakLine("production", ContractBreak{
			CheckedResource: consumer,
			Reason:          "provider_resource_not_deployed_in_environment",
			Details:         map[string]string{"deployedEnvironments": "staging, qa"},
		})

		assert.Equal(t, `provider is not deployed in "production" (deployed in: staging, qa)`, line)
	})

	t.Run("provider_resource_not_deployed_in_environment without deployedEnvironments", func(t *testing.T) {
		line := formatBreakLine("production", ContractBreak{
			CheckedResource: consumer,
			Reason:          "provider_resource_not_deployed_in_environment",
		})

		assert.Equal(t, `provider is not deployed in "production"`, line)
	})

	t.Run("property_missing_in_provider", func(t *testing.T) {
		line := formatBreakLine("production", ContractBreak{
			CheckedResource:     consumer,
			CounterpartResource: provider,
			Reason:              "property_missing_in_provider",
			Details:             map[string]string{"property": "amount"},
		})

		assert.Equal(t, `GET /payments/* response 200: property "amount" is missing in provider`, line)
	})

	t.Run("property_missing_in_consumer", func(t *testing.T) {
		line := formatBreakLine("production", ContractBreak{
			CheckedResource:     consumer,
			CounterpartResource: provider,
			Reason:              "property_missing_in_consumer",
			Details:             map[string]string{"property": "amount"},
		})

		assert.Equal(t, `GET /payments/* response 200: property "amount" is missing in consumer`, line)
	})

	t.Run("property_optional_in_provider_required_in_consumer", func(t *testing.T) {
		line := formatBreakLine("production", ContractBreak{
			CheckedResource:     consumer,
			CounterpartResource: provider,
			Reason:              "property_optional_in_provider_required_in_consumer",
			Details:             map[string]string{"property": "amount"},
		})

		assert.Equal(t, `GET /payments/* response 200: property "amount" is optional in provider but required in consumer`, line)
	})

	t.Run("property_optional_in_consumer_required_in_provider", func(t *testing.T) {
		line := formatBreakLine("production", ContractBreak{
			CheckedResource:     consumer,
			CounterpartResource: provider,
			Reason:              "property_optional_in_consumer_required_in_provider",
			Details:             map[string]string{"property": "amount"},
		})

		assert.Equal(t, `GET /payments/* response 200: property "amount" is optional in consumer but required in provider`, line)
	})

	t.Run("property_type_mismatch maps types by role when the checked side is the provider", func(t *testing.T) {
		line := formatBreakLine("production", ContractBreak{
			CheckedResource:     provider,
			CounterpartResource: consumer,
			Reason:              "property_type_mismatch",
			Details: map[string]string{
				"property":                "amount",
				"checkedPropertyType":     "number",
				"counterpartPropertyType": "string",
			},
		})

		assert.Equal(t, `GET /payments/* response 200: property "amount" type mismatch — consumer has string, provider has number`, line)
	})

	t.Run("property_type_mismatch maps types by role when the checked side is the consumer", func(t *testing.T) {
		line := formatBreakLine("production", ContractBreak{
			CheckedResource:     consumer,
			CounterpartResource: provider,
			Reason:              "property_type_mismatch",
			Details: map[string]string{
				"property":                "amount",
				"checkedPropertyType":     "string",
				"counterpartPropertyType": "number",
			},
		})

		assert.Equal(t, `GET /payments/* response 200: property "amount" type mismatch — consumer has string, provider has number`, line)
	})

	t.Run("unknown reason falls back to the verbatim reason with details", func(t *testing.T) {
		line := formatBreakLine("production", ContractBreak{
			CheckedResource: consumer,
			Reason:          "some_future_reason",
			Details:         map[string]string{"property": "amount", "extra": "x"},
		})

		assert.Equal(t, `some_future_reason (extra: x, property: amount)`, line)
	})
}
