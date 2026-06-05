package compatibility_checker

import (
	"strings"

	"github.com/contracttesting/broker/internal/model"
)

type BreakingReason string

const (
	ReasonProviderResourceNotFound                 BreakingReason = "provider_resource_not_found"
	ReasonMissingInProvider                        BreakingReason = "missing_in_provider"
	ReasonMissingInConsumer                        BreakingReason = "missing_in_consumer"
	ReasonTypeMismatch                             BreakingReason = "type_mismatch"
	ReasonOptionalInProviderRequiredInConsumer     BreakingReason = "optional_in_provider_required_in_consumer"
	ReasonOptionalInConsumerRequiredInProvider     BreakingReason = "optional_in_consumer_required_in_provider"
	ReasonProviderResourceNotDeployedInEnvironment BreakingReason = "provider_resource_not_deployed_in_environment"
)

const (
	detailKeyProperty                = "property"
	detailKeyCheckedPropertyType     = "checked_property_type"
	detailKeyCounterpartPropertyType = "counterpart_property_type"
	detailKeyDeployedEnvironments    = "deployed_environments"
)

type BreakingChange struct {
	CheckedResource     *model.Resource   `json:"checked_resource"`
	CounterpartResource *model.Resource   `json:"counterpart_resource,omitempty"`
	Reason              BreakingReason    `json:"reason"`
	Details             map[string]string `json:"details,omitempty"`
}

func (b *BreakingChange) consumerResource() *model.Resource {
	if b.CheckedResource.IsConsumer() {
		return b.CheckedResource
	}

	return b.CounterpartResource
}

func (b *BreakingChange) ConsumerName() string {
	return b.consumerResource().ParticipantName()
}

func (b *BreakingChange) ProviderName() string {
	return b.consumerResource().ConsumedProvider
}

func NewBreakingChange(
	checked *model.Resource,
	counterpart *model.Resource,
	reason BreakingReason,
) BreakingChange {
	return BreakingChange{
		CheckedResource:     checked,
		CounterpartResource: counterpart,
		Reason:              reason,
	}
}

func NewPropertyBreakChange(
	checked *model.Resource,
	counterpart *model.Resource,
	reason BreakingReason,
	property string,
) BreakingChange {
	details := map[string]string{detailKeyProperty: property}

	if reason == ReasonTypeMismatch {
		details[detailKeyCheckedPropertyType] = checked.Properties[property].Type
		details[detailKeyCounterpartPropertyType] = counterpart.Properties[property].Type
	}

	return BreakingChange{
		CheckedResource:     checked,
		CounterpartResource: counterpart,
		Reason:              reason,
		Details:             details,
	}
}

func NewProviderNotDeployedBreakingChange(
	consumer *model.Resource,
	deployedEnvironments []string,
) BreakingChange {
	var details map[string]string

	if len(deployedEnvironments) > 0 {
		details = map[string]string{
			detailKeyDeployedEnvironments: strings.Join(deployedEnvironments, ", "),
		}
	}

	return BreakingChange{
		CheckedResource: consumer,
		Reason:          ReasonProviderResourceNotDeployedInEnvironment,
		Details:         details,
	}
}
