package compatibility_checker_test

import (
	"testing"

	"github.com/contracttesting/broker/internal/compatibility_checker"
	"github.com/contracttesting/broker/internal/model"
	"github.com/guregu/null"
	"github.com/stretchr/testify/assert"
)

func consumerResource() *model.CounterpartResource {
	return &model.CounterpartResource{
		Direction:          model.Consumes,
		Interaction:        model.RestResponse,
		ConsumedProvider:   null.StringFrom("api"),
		Endpoint:           "/things",
		Method:             "get",
		ResponseStatusCode: null.StringFrom("200"),
		ParticipantName:    "front",
		Properties: map[string]model.Property{
			"$.id": {Path: "$.id", Type: "integer"},
		},
	}
}

func providerResource() *model.CounterpartResource {
	return &model.CounterpartResource{
		Direction:          model.Provides,
		Interaction:        model.RestResponse,
		Endpoint:           "/things",
		Method:             "get",
		ResponseStatusCode: null.StringFrom("200"),
		ParticipantName:    "api",
		Properties: map[string]model.Property{
			"$.id": {Path: "$.id", Type: "string"},
		},
	}
}

func TestProviderResourceNotFound(t *testing.T) {
	consumer := &model.CounterpartResource{
		Direction:        model.Consumes,
		ConsumedProvider: null.StringFrom("accounts"),
		ParticipantName:  "front",
	}

	change := compatibility_checker.NewContractBreakingChange(
		consumer,
		nil,
		compatibility_checker.ReasonProviderResourceNotFound,
	)

	assert.Equal(t, compatibility_checker.ReasonProviderResourceNotFound, change.Reason)
	assert.Same(t, consumer, change.CheckedResource)
	assert.Nil(t, change.CounterpartResource)
	assert.Nil(t, change.Details)
	assert.Equal(t, "front", change.ConsumerName())
	assert.Equal(t, "accounts", change.ProviderName())
}

func TestPropertyBreakDetails(t *testing.T) {
	consumerResponse := consumerResource()
	providerResponse := providerResource()
	consumerRequest := &model.CounterpartResource{
		Direction:        model.Consumes,
		Interaction:      model.RestRequest,
		ConsumedProvider: null.StringFrom("api"),
		Endpoint:         "/things",
		Method:           "post",
		ParticipantName:  "front",
	}
	providerRequest := &model.CounterpartResource{
		Direction:       model.Provides,
		Interaction:     model.RestRequest,
		Endpoint:        "/things",
		Method:          "post",
		ParticipantName: "api",
	}

	typeMismatch := compatibility_checker.NewPropertyBreakChange(
		consumerResponse, providerResponse, compatibility_checker.ReasonPropertyTypeMismatch, "$.id",
	)
	missingInProvider := compatibility_checker.NewPropertyBreakChange(
		consumerResponse, providerResponse, compatibility_checker.ReasonPropertyMissingInProvider, "$.name",
	)
	optionalInProvider := compatibility_checker.NewPropertyBreakChange(
		consumerResponse, providerResponse, compatibility_checker.ReasonPropertyOptionalInProviderRequiredInConsumer, "$.id",
	)
	missingInConsumer := compatibility_checker.NewPropertyBreakChange(
		consumerRequest, providerRequest, compatibility_checker.ReasonPropertyMissingInConsumer, "$.user",
	)
	optionalInConsumer := compatibility_checker.NewPropertyBreakChange(
		consumerRequest, providerRequest, compatibility_checker.ReasonPropertyOptionalInConsumerRequiredInProvider, "$.flag",
	)

	assert.Equal(t, compatibility_checker.ReasonPropertyTypeMismatch, typeMismatch.Reason)
	assert.Equal(t, compatibility_checker.ReasonPropertyMissingInProvider, missingInProvider.Reason)
	assert.Equal(t, compatibility_checker.ReasonPropertyOptionalInProviderRequiredInConsumer, optionalInProvider.Reason)
	assert.Equal(t, compatibility_checker.ReasonPropertyMissingInConsumer, missingInConsumer.Reason)
	assert.Equal(t, compatibility_checker.ReasonPropertyOptionalInConsumerRequiredInProvider, optionalInConsumer.Reason)

	assert.Equal(t, map[string]string{
		"property":                  "$.id",
		"checked_property_type":     "integer",
		"counterpart_property_type": "string",
	}, typeMismatch.Details)

	assert.Equal(t, map[string]string{"property": "$.name"}, missingInProvider.Details)
	assert.Equal(t, map[string]string{"property": "$.id"}, optionalInProvider.Details)
	assert.Equal(t, map[string]string{"property": "$.user"}, missingInConsumer.Details)
	assert.Equal(t, map[string]string{"property": "$.flag"}, optionalInConsumer.Details)

	assert.Equal(t, "front", typeMismatch.ConsumerName())
	assert.Equal(t, "api", typeMismatch.ProviderName())
	assert.Equal(t, "front", missingInConsumer.ConsumerName())
	assert.Equal(t, "api", missingInConsumer.ProviderName())
}

func TestTypeMismatchTypesFollowCheckedSide(t *testing.T) {
	consumerRes := consumerResource()
	providerRes := providerResource()

	consumerChecked := compatibility_checker.NewPropertyBreakChange(
		consumerRes, providerRes, compatibility_checker.ReasonPropertyTypeMismatch, "$.id",
	)
	providerChecked := compatibility_checker.NewPropertyBreakChange(
		providerRes, consumerRes, compatibility_checker.ReasonPropertyTypeMismatch, "$.id",
	)

	assert.Equal(t, map[string]string{
		"property":                  "$.id",
		"checked_property_type":     "integer",
		"counterpart_property_type": "string",
	}, consumerChecked.Details)
	assert.Equal(t, map[string]string{
		"property":                  "$.id",
		"checked_property_type":     "string",
		"counterpart_property_type": "integer",
	}, providerChecked.Details)

	assert.Same(t, providerRes, providerChecked.CheckedResource)
	assert.Same(t, consumerRes, providerChecked.CounterpartResource)

	assert.Equal(t, "front", consumerChecked.ConsumerName())
	assert.Equal(t, "api", consumerChecked.ProviderName())
	assert.Equal(t, "front", providerChecked.ConsumerName())
	assert.Equal(t, "api", providerChecked.ProviderName())
}

func TestProviderNotDeployedDetails(t *testing.T) {
	consumer := &model.CounterpartResource{
		Direction:        model.Consumes,
		ConsumedProvider: null.StringFrom("accounts"),
		ParticipantName:  "front",
	}

	deployed := compatibility_checker.NewProviderNotDeployedBreakingChange(
		consumer, []string{"staging", "prod"},
	)
	nowhere := compatibility_checker.NewProviderNotDeployedBreakingChange(
		consumer, nil,
	)

	assert.Equal(t, compatibility_checker.ReasonProviderResourceNotDeployedInEnvironment, deployed.Reason)
	assert.Equal(t, map[string]string{"deployed_environments": "staging, prod"}, deployed.Details)
	assert.Equal(t, "front", deployed.ConsumerName())
	assert.Equal(t, "accounts", deployed.ProviderName())

	assert.Equal(t, compatibility_checker.ReasonProviderResourceNotDeployedInEnvironment, nowhere.Reason)
	assert.Nil(t, nowhere.Details)
}

func TestReportAppendCollectsBreaksAsList(t *testing.T) {
	consumerRes := consumerResource()
	providerRes := providerResource()

	consumerChecked := compatibility_checker.NewPropertyBreakChange(
		consumerRes, providerRes, compatibility_checker.ReasonPropertyTypeMismatch, "$.id",
	)
	providerChecked := compatibility_checker.NewPropertyBreakChange(
		providerRes, consumerRes, compatibility_checker.ReasonPropertyTypeMismatch, "$.id",
	)

	report := compatibility_checker.NewCompatibilityReport()
	report.AppendContractBreakChange(consumerChecked)
	report.AppendContractBreakChange(providerChecked)

	assert.Len(t, report.Breaks, 2)
}
