package compatibility_checker

import (
	"strings"

	"github.com/contracttesting/broker/internal/model"
)

type BreakingReason string

const (
	ReasonProviderResourceNotFound                     BreakingReason = "provider_resource_not_found"
	ReasonPropertyMissingInProvider                    BreakingReason = "property_missing_in_provider"
	ReasonPropertyMissingInConsumer                    BreakingReason = "property_missing_in_consumer"
	ReasonPropertyTypeMismatch                         BreakingReason = "property_type_mismatch"
	ReasonPropertyOptionalInProviderRequiredInConsumer BreakingReason = "property_optional_in_provider_required_in_consumer"
	ReasonPropertyOptionalInConsumerRequiredInProvider BreakingReason = "property_optional_in_consumer_required_in_provider"
	ReasonProviderResourceNotDeployedInEnvironment     BreakingReason = "provider_resource_not_deployed_in_environment"
)

type ContractBreakingChange struct {
	CheckedResource     *model.PersistedResource `json:"-"`
	CounterpartResource *model.PersistedResource `json:"-"`
	Reason              BreakingReason           `json:"reason"`
	Details             map[string]string        `json:"details,omitempty"`
}

func (b *ContractBreakingChange) ConsumerResource() *model.PersistedResource {
	if b.CheckedResource.IsConsumer() {
		return b.CheckedResource
	}

	return b.CounterpartResource
}

func (b *ContractBreakingChange) ConsumerName() string {
	return b.ConsumerResource().ParticipantName
}

func (b *ContractBreakingChange) ProviderName() string {
	return b.ConsumerResource().ConsumedProvider.String
}

func NewContractBreakingChange(
	checkedResource *model.PersistedResource,
	counterpartResource *model.PersistedResource,
	reason BreakingReason,
) ContractBreakingChange {
	return ContractBreakingChange{
		CheckedResource:     checkedResource,
		CounterpartResource: counterpartResource,
		Reason:              reason,
	}
}

func NewProviderResourceNotFound(checkedResource *model.PersistedResource) ContractBreakingChange {
	return ContractBreakingChange{
		CheckedResource: checkedResource,
		Reason:          ReasonProviderResourceNotFound,
	}
}

func NewPropertyBreakChange(
	checkedResource *model.PersistedResource,
	counterpartResource *model.PersistedResource,
	reason BreakingReason,
	propertyPath string,
) ContractBreakingChange {
	details := map[string]string{"property": propertyPath}

	if reason == ReasonPropertyTypeMismatch {
		consumerResource, providerResource := checkedResource, counterpartResource
		if !checkedResource.IsConsumer() {
			consumerResource, providerResource = counterpartResource, checkedResource
		}

		details["consumerPropertyType"] = consumerResource.Properties[propertyPath].Type
		details["providerPropertyType"] = providerResource.Properties[propertyPath].Type
	}

	return ContractBreakingChange{
		CheckedResource:     checkedResource,
		CounterpartResource: counterpartResource,
		Reason:              reason,
		Details:             details,
	}
}

func NewProviderResourceNotDeployed(
	consumerResource *model.PersistedResource,
	deployedEnvironments []string,
) ContractBreakingChange {
	var details map[string]string

	if len(deployedEnvironments) > 0 {
		details = map[string]string{
			"deployedEnvironments": strings.Join(deployedEnvironments, ", "),
		}
	}

	return ContractBreakingChange{
		CheckedResource: consumerResource,
		Reason:          ReasonProviderResourceNotDeployedInEnvironment,
		Details:         details,
	}
}
