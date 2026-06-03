package compatibility_checker_test

import (
	"testing"

	"github.com/contracttesting/broker/internal/compatibility_checker"
	"github.com/contracttesting/broker/internal/model"
	"github.com/stretchr/testify/assert"
)

// A provider-resource-not-found break must carry a human-readable message.
// It used to be appended as a bare BreakingChange struct literal that skipped
// NewBreakingChange (and thus humanReadable()), so the CLI rendered empty
// "  - " bullets for every such break.
func TestProviderResourceNotFoundHasHumanReadable(t *testing.T) {
	consumer := &model.Resource{
		ConsumedProvider: "accounts",
		Participant:      &model.Participant{Name: "front"},
	}

	change := compatibility_checker.NewBreakingChange(
		consumer,
		nil,
		compatibility_checker.ReasonProviderResourceNotFound,
		"",
	)

	assert.NotEmpty(t, change.HumanReadable)
	assert.Contains(t, change.HumanReadable, "was found")
	assert.Equal(t, "front", change.ConsumerName)
	assert.Equal(t, "accounts", change.ProviderName)
	assert.Empty(t, change.ConsumerType)
	assert.Empty(t, change.ProviderType)
}

// The message describes the missing HTTP operation, upper-casing the method and,
// for a response, including the expected status. The provider is conveyed by the
// CLI's grouping, not the message.
func TestProviderResourceNotFoundMessage(t *testing.T) {
	request := &model.Resource{
		Kind:             model.RestRequest,
		ConsumedProvider: "accounts",
		Endpoint:         "/accounts",
		Method:           "post",
		Participant:      &model.Participant{Name: "front"},
	}
	response := &model.Resource{
		Kind:               model.RestResponse,
		ConsumedProvider:   "accounts",
		Endpoint:           "/accounts",
		Method:             "post",
		ResponseStatusCode: "201",
		Participant:        &model.Participant{Name: "front"},
	}

	requestChange := compatibility_checker.NewBreakingChange(
		request, nil, compatibility_checker.ReasonProviderResourceNotFound, "",
	)
	responseChange := compatibility_checker.NewBreakingChange(
		response, nil, compatibility_checker.ReasonProviderResourceNotFound, "",
	)

	assert.Equal(t, `No POST /accounts (request) was found`, requestChange.HumanReadable)
	assert.Equal(t, `No POST /accounts (response 201) was found`, responseChange.HumanReadable)
}

// Property breaks name the HTTP operation they occur on, using the same
// upper-cased "METHOD /path (...)" form as resource-not-found breaks, so a
// reader can tell which endpoint each property problem belongs to.
func TestPropertyBreakMessagesIncludeOperation(t *testing.T) {
	consumerResponse := &model.Resource{
		Kind:               model.RestResponse,
		Endpoint:           "/things",
		Method:             "get",
		ResponseStatusCode: "200",
		Participant:        &model.Participant{Name: "front"},
		Properties: map[string]model.Property{
			"root.id": {Path: "root.id", Type: "integer"},
		},
	}
	providerResponse := &model.Resource{
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
		Kind:        model.RestRequest,
		Endpoint:    "/things",
		Method:      "post",
		Participant: &model.Participant{Name: "front"},
	}
	providerRequest := &model.Resource{
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

	assert.Equal(t,
		`Property root.id type mismatch on GET /things (response 200), provider api expects string but consumer front expects integer`,
		typeMismatch.HumanReadable)
	assert.Equal(t,
		`Property root.name is missing in provider api on GET /things (response 200)`,
		missingInProvider.HumanReadable)
	assert.Equal(t,
		`Property root.id is optional in provider api but required in consumer front on GET /things (response 200)`,
		optionalInProvider.HumanReadable)
	assert.Equal(t,
		`Property root.user is missing in consumer front on POST /things (request)`,
		missingInConsumer.HumanReadable)
	assert.Equal(t,
		`Property root.flag is optional in consumer front but required in provider api on POST /things (request)`,
		optionalInConsumer.HumanReadable)

	// Every break carries the consumer name. Type-mismatch breaks additionally
	// carry the consumer/provider property types; other reasons leave them empty.
	assert.Equal(t, "front", typeMismatch.ConsumerName)
	assert.Equal(t, "integer", typeMismatch.ConsumerType)
	assert.Equal(t, "string", typeMismatch.ProviderType)

	assert.Equal(t, "front", missingInProvider.ConsumerName)
	assert.Empty(t, missingInProvider.ConsumerType)
	assert.Empty(t, missingInProvider.ProviderType)

	assert.Equal(t, "front", missingInConsumer.ConsumerName)
	assert.Empty(t, missingInConsumer.ConsumerType)
	assert.Empty(t, missingInConsumer.ProviderType)
}
