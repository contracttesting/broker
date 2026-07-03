package compatibility_checker_test

import (
	"testing"

	"github.com/contracttesting/broker/internal/compatibility_checker"
	"github.com/contracttesting/broker/internal/model"
	"github.com/guregu/null"
	"github.com/stretchr/testify/assert"
)

func consumerResource() *model.PersistedResource {
	return &model.PersistedResource{
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

func providerResource() *model.PersistedResource {
	return &model.PersistedResource{
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
	consumer := &model.PersistedResource{
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
	consumerRequest := &model.PersistedResource{
		Direction:        model.Consumes,
		Interaction:      model.RestRequest,
		ConsumedProvider: null.StringFrom("api"),
		Endpoint:         "/things",
		Method:           "post",
		ParticipantName:  "front",
	}
	providerRequest := &model.PersistedResource{
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
		"property":                "$.id",
		"checkedPropertyType":     "integer",
		"counterpartPropertyType": "string",
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
		"property":                "$.id",
		"checkedPropertyType":     "integer",
		"counterpartPropertyType": "string",
	}, consumerChecked.Details)
	assert.Equal(t, map[string]string{
		"property":                "$.id",
		"checkedPropertyType":     "string",
		"counterpartPropertyType": "integer",
	}, providerChecked.Details)

	assert.Same(t, providerRes, providerChecked.CheckedResource)
	assert.Same(t, consumerRes, providerChecked.CounterpartResource)

	assert.Equal(t, "front", consumerChecked.ConsumerName())
	assert.Equal(t, "api", consumerChecked.ProviderName())
	assert.Equal(t, "front", providerChecked.ConsumerName())
	assert.Equal(t, "api", providerChecked.ProviderName())
}

func TestProviderNotDeployedDetails(t *testing.T) {
	consumer := &model.PersistedResource{
		Direction:        model.Consumes,
		ConsumedProvider: null.StringFrom("accounts"),
		ParticipantName:  "front",
	}

	deployed := compatibility_checker.NewProviderResourceNotDeployed(
		consumer, []string{"staging", "prod"},
	)
	nowhere := compatibility_checker.NewProviderResourceNotDeployed(
		consumer, nil,
	)

	assert.Equal(t, compatibility_checker.ReasonProviderResourceNotDeployedInEnvironment, deployed.Reason)
	assert.Equal(t, map[string]string{"deployedEnvironments": "staging, prod"}, deployed.Details)
	assert.Equal(t, "front", deployed.ConsumerName())
	assert.Equal(t, "accounts", deployed.ProviderName())

	assert.Equal(t, compatibility_checker.ReasonProviderResourceNotDeployedInEnvironment, nowhere.Reason)
	assert.Nil(t, nowhere.Details)
}

func TestReportAppendMergesResultsPerCounterpart(t *testing.T) {
	consumerRes := consumerResource()
	providerRes := providerResource()

	consumerChecked := compatibility_checker.NewPropertyBreakChange(
		consumerRes, providerRes, compatibility_checker.ReasonPropertyTypeMismatch, "$.id",
	)
	providerChecked := compatibility_checker.NewPropertyBreakChange(
		providerRes, consumerRes, compatibility_checker.ReasonPropertyTypeMismatch, "$.id",
	)

	report := compatibility_checker.NewContractCompatibilityReport("unknown", "unknown", "unknwonw")

	withoutIdentity := compatibility_checker.NewIncompatibleItem()
	withoutIdentity.AppendContractBreakChange(consumerChecked)
	report.AppendResult("api", withoutIdentity)

	withIdentity := compatibility_checker.NewIncompatibleItem()
	withIdentity.IncompatibleCounterpart = compatibility_checker.IncompatibleCounterpart{
		ParticipantID:      7,
		ParticipantVersion: null.StringFrom("v1"),
	}
	withIdentity.AppendContractBreakChange(providerChecked)
	report.AppendResult("api", withIdentity)

	auth := compatibility_checker.NewIncompatibleItem()
	auth.IncompatibleCounterpart = compatibility_checker.IncompatibleCounterpart{
		ParticipantID:      9,
		ParticipantVersion: null.StringFrom("v2"),
	}
	report.AppendResult("auth", auth)

	assert.Len(t, report.Results, 2)

	apiResult := report.Results["api"]
	assert.Len(t, apiResult.Breaks, 2)
	assert.False(t, apiResult.Deployable)
	assert.Equal(t, int64(7), apiResult.IncompatibleCounterpart.ParticipantID)
	assert.Equal(t, "api", apiResult.IncompatibleCounterpart.ParticipantName)
	assert.Equal(t, null.StringFrom("v1"), apiResult.IncompatibleCounterpart.ParticipantVersion)

	authResult := report.Results["auth"]
	assert.Empty(t, authResult.Breaks)
	assert.True(t, authResult.Deployable)
	assert.Equal(t, int64(9), authResult.IncompatibleCounterpart.ParticipantID)
	assert.Equal(t, "auth", authResult.IncompatibleCounterpart.ParticipantName)
	assert.Equal(t, null.StringFrom("v2"), authResult.IncompatibleCounterpart.ParticipantVersion)
}
