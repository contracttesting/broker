package compatibility_checker_test

import (
	"testing"

	"github.com/contracttesting/broker/internal/compatibility_checker"
	"github.com/contracttesting/broker/internal/model"
	"github.com/stretchr/testify/assert"
)

// A provider-resource-not-found break carries the checked consumer, no
// counterpart, and no Details: there is no property and no counterpart side to
// describe. Roles still resolve from the checked consumer, so its participant
// and consumed provider surface as the consumer/provider names.
func TestProviderResourceNotFound(t *testing.T) {
	consumer := &model.Resource{
		Direction:        model.Consumes,
		ConsumedProvider: "accounts",
		Participant:      &model.Participant{Name: "front"},
	}

	change := compatibility_checker.NewBreakingChange(
		consumer,
		nil,
		compatibility_checker.ReasonProviderResourceNotFound,
		"",
	)

	assert.Equal(t, compatibility_checker.ReasonProviderResourceNotFound, change.Reason)
	assert.Same(t, consumer, change.CheckedResource)
	assert.Nil(t, change.CounterpartResource)
	assert.Nil(t, change.Details)
	assert.Equal(t, "front", change.ConsumerName())
	assert.Equal(t, "accounts", change.ProviderName())
}

// Property-level breaks record the property path in Details. Type mismatches
// additionally record both sides' property types, always keyed by role
// (consumer_type / provider_type) and never by argument position. Reasons
// round-trip unchanged. These are all consumer-checked breaks.
func TestPropertyBreakDetails(t *testing.T) {
	consumerResponse := &model.Resource{
		Direction:          model.Consumes,
		Kind:               model.RestResponse,
		ConsumedProvider:   "api",
		Endpoint:           "/things",
		Method:             "get",
		ResponseStatusCode: "200",
		Participant:        &model.Participant{Name: "front"},
		Properties: map[string]model.Property{
			"root.id": {Path: "root.id", Type: "integer"},
		},
	}
	providerResponse := &model.Resource{
		Direction:          model.Provides,
		Kind:               model.RestResponse,
		Endpoint:           "/things",
		Method:             "get",
		ResponseStatusCode: "200",
		Participant:        &model.Participant{Name: "api"},
		Properties: map[string]model.Property{
			"root.id": {Path: "root.id", Type: "string"},
		},
	}
	consumerRequest := &model.Resource{
		Direction:        model.Consumes,
		Kind:             model.RestRequest,
		ConsumedProvider: "api",
		Endpoint:         "/things",
		Method:           "post",
		Participant:      &model.Participant{Name: "front"},
	}
	providerRequest := &model.Resource{
		Direction:   model.Provides,
		Kind:        model.RestRequest,
		Endpoint:    "/things",
		Method:      "post",
		Participant: &model.Participant{Name: "api"},
	}

	typeMismatch := compatibility_checker.NewBreakingChange(
		consumerResponse, providerResponse, compatibility_checker.ReasonTypeMismatch, "root.id",
	)
	missingInProvider := compatibility_checker.NewBreakingChange(
		consumerResponse, providerResponse, compatibility_checker.ReasonMissingInProvider, "root.name",
	)
	optionalInProvider := compatibility_checker.NewBreakingChange(
		consumerResponse, providerResponse, compatibility_checker.ReasonOptionalInProviderRequiredInConsumer, "root.id",
	)
	missingInConsumer := compatibility_checker.NewBreakingChange(
		consumerRequest, providerRequest, compatibility_checker.ReasonMissingInConsumer, "root.user",
	)
	optionalInConsumer := compatibility_checker.NewBreakingChange(
		consumerRequest, providerRequest, compatibility_checker.ReasonOptionalInConsumerRequiredInProvider, "root.flag",
	)

	// Reasons round-trip.
	assert.Equal(t, compatibility_checker.ReasonTypeMismatch, typeMismatch.Reason)
	assert.Equal(t, compatibility_checker.ReasonMissingInProvider, missingInProvider.Reason)
	assert.Equal(t, compatibility_checker.ReasonOptionalInProviderRequiredInConsumer, optionalInProvider.Reason)
	assert.Equal(t, compatibility_checker.ReasonMissingInConsumer, missingInConsumer.Reason)
	assert.Equal(t, compatibility_checker.ReasonOptionalInConsumerRequiredInProvider, optionalInConsumer.Reason)

	// Type mismatch carries property + both role-keyed types.
	assert.Equal(t, map[string]string{
		"property":      "root.id",
		"consumer_type": "integer",
		"provider_type": "string",
	}, typeMismatch.Details)

	// Non-type-mismatch property breaks carry only the property path.
	assert.Equal(t, map[string]string{"property": "root.name"}, missingInProvider.Details)
	assert.Equal(t, map[string]string{"property": "root.id"}, optionalInProvider.Details)
	assert.Equal(t, map[string]string{"property": "root.user"}, missingInConsumer.Details)
	assert.Equal(t, map[string]string{"property": "root.flag"}, optionalInConsumer.Details)

	// Names resolve from the consumes side for every consumer-checked break.
	assert.Equal(t, "front", typeMismatch.ConsumerName())
	assert.Equal(t, "api", typeMismatch.ProviderName())
	assert.Equal(t, "front", missingInConsumer.ConsumerName())
	assert.Equal(t, "api", missingInConsumer.ProviderName())
}

// Role resolution is by direction, not argument order. The same logical type
// mismatch, constructed consumer-checked and provider-checked (swapped
// arguments), yields identical Details: consumer_type always comes from the
// consumes-side property type, provider_type from the provides side. The
// provider-checked construction also exercises a checked side with
// Direction: Provides and asserts consumer/provider names resolve correctly.
func TestTypeMismatchRoleResolutionIsDirectionBased(t *testing.T) {
	consumerRes := &model.Resource{
		Direction:          model.Consumes,
		Kind:               model.RestResponse,
		ConsumedProvider:   "api",
		Endpoint:           "/things",
		Method:             "get",
		ResponseStatusCode: "200",
		Participant:        &model.Participant{Name: "front"},
		Properties: map[string]model.Property{
			"root.id": {Path: "root.id", Type: "integer"},
		},
	}
	providerRes := &model.Resource{
		Direction:          model.Provides,
		Kind:               model.RestResponse,
		Endpoint:           "/things",
		Method:             "get",
		ResponseStatusCode: "200",
		Participant:        &model.Participant{Name: "api"},
		Properties: map[string]model.Property{
			"root.id": {Path: "root.id", Type: "string"},
		},
	}

	consumerChecked := compatibility_checker.NewBreakingChange(
		consumerRes, providerRes, compatibility_checker.ReasonTypeMismatch, "root.id",
	)
	providerChecked := compatibility_checker.NewBreakingChange(
		providerRes, consumerRes, compatibility_checker.ReasonTypeMismatch, "root.id",
	)

	expectedDetails := map[string]string{
		"property":      "root.id",
		"consumer_type": "integer",
		"provider_type": "string",
	}
	assert.Equal(t, expectedDetails, consumerChecked.Details,
		"consumer_type must come from the consumes-side property type")
	assert.Equal(t, expectedDetails, providerChecked.Details,
		"role resolution is by direction: swapping arguments must not swap types")

	// The provider-checked break stores the provider as the checked resource.
	assert.Same(t, providerRes, providerChecked.CheckedResource)
	assert.Same(t, consumerRes, providerChecked.CounterpartResource)

	// Both directions resolve to the same consumer/provider names.
	assert.Equal(t, "front", consumerChecked.ConsumerName())
	assert.Equal(t, "api", consumerChecked.ProviderName())
	assert.Equal(t, "front", providerChecked.ConsumerName())
	assert.Equal(t, "api", providerChecked.ProviderName())
}

// A provider-not-deployed break joins the deployed environments into a single
// Details entry. With no environments it carries no Details at all.
func TestProviderNotDeployedDetails(t *testing.T) {
	consumer := &model.Resource{
		Direction:        model.Consumes,
		ConsumedProvider: "accounts",
		Participant:      &model.Participant{Name: "front"},
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

// Append keys breaks by the resolved consumer name regardless of whether the
// break was consumer-checked or provider-checked, so both land under the same
// consumer bucket.
func TestReportAppendKeysByConsumerName(t *testing.T) {
	consumerRes := &model.Resource{
		Direction:          model.Consumes,
		Kind:               model.RestResponse,
		ConsumedProvider:   "api",
		Endpoint:           "/things",
		Method:             "get",
		ResponseStatusCode: "200",
		Participant:        &model.Participant{Name: "front"},
		Properties: map[string]model.Property{
			"root.id": {Path: "root.id", Type: "integer"},
		},
	}
	providerRes := &model.Resource{
		Direction:          model.Provides,
		Kind:               model.RestResponse,
		Endpoint:           "/things",
		Method:             "get",
		ResponseStatusCode: "200",
		Participant:        &model.Participant{Name: "api"},
		Properties: map[string]model.Property{
			"root.id": {Path: "root.id", Type: "string"},
		},
	}

	consumerChecked := compatibility_checker.NewBreakingChange(
		consumerRes, providerRes, compatibility_checker.ReasonTypeMismatch, "root.id",
	)
	providerChecked := compatibility_checker.NewBreakingChange(
		providerRes, consumerRes, compatibility_checker.ReasonTypeMismatch, "root.id",
	)

	report := compatibility_checker.NewCompatibilityReport()
	report.Append(consumerChecked)
	report.Append(providerChecked)

	assert.Len(t, report.Breaks, 1)
	assert.Len(t, report.Breaks["front"], 2)
}
