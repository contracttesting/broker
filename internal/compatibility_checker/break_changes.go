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
	CheckedResource     *model.CounterpartResource `json:"checked_resource,omitempty"`
	CounterpartResource *model.CounterpartResource `json:"counterpart_resource,omitempty"`
	Reason              BreakingReason             `json:"reason"`
	Details             map[string]string          `json:"details,omitempty"`
}

func (b *ContractBreakingChange) ConsumerResource() *model.CounterpartResource {
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
	checkedResource *model.CounterpartResource,
	counterpartResource *model.CounterpartResource,
	reason BreakingReason,
) ContractBreakingChange {
	return ContractBreakingChange{
		CheckedResource:     checkedResource,
		CounterpartResource: counterpartResource,
		Reason:              reason,
	}
}

func NewPropertyBreakChange(
	checkedResource *model.CounterpartResource,
	counterpartResource *model.CounterpartResource,
	reason BreakingReason,
	propertyPath string,
) ContractBreakingChange {
	details := map[string]string{"property": propertyPath}

	if reason == ReasonPropertyTypeMismatch {
		details["checked_property_type"] = checkedResource.Properties[propertyPath].Type
		details["counterpart_property_type"] = counterpartResource.Properties[propertyPath].Type
	}

	return ContractBreakingChange{
		CheckedResource:     checkedResource,
		CounterpartResource: counterpartResource,
		Reason:              reason,
		Details:             details,
	}
}

func NewProviderNotDeployedBreakingChange(
	consumerResource *model.CounterpartResource,
	deployedEnvironments []string,
) ContractBreakingChange {
	var details map[string]string

	if len(deployedEnvironments) > 0 {
		details = map[string]string{
			"deployed_environments": strings.Join(deployedEnvironments, ", "),
		}
	}

	return ContractBreakingChange{
		CheckedResource: consumerResource,
		Reason:          ReasonProviderResourceNotDeployedInEnvironment,
		Details:         details,
	}
}
