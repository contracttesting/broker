package compatibility_checker

import (
	"strconv"
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
	ReasonPropertyNotMatchingAnyVariant                BreakingReason = "property_not_matching_any_variant"
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
	consumerResource, providerResource := checkedResource, counterpartResource
	if !checkedResource.IsConsumer() {
		consumerResource, providerResource = counterpartResource, checkedResource
	}

	details := map[string]string{
		"property":     propertyPath,
		"consumerName": consumerResource.ParticipantName,
		"providerName": consumerResource.ConsumedProvider.String,
	}

	if reason == ReasonPropertyTypeMismatch {
		details["consumerPropertyType"] = typeToken(consumerResource, propertyPath)
		details["providerPropertyType"] = typeToken(providerResource, propertyPath)
	} else {
		typed := consumerResource
		if _, exists := consumerResource.Properties[propertyPath]; !exists {
			typed = providerResource
		}
		details["propertyType"] = typeToken(typed, propertyPath)
	}

	return ContractBreakingChange{
		CheckedResource:     checkedResource,
		CounterpartResource: counterpartResource,
		Reason:              reason,
		Details:             details,
	}
}

// NewUnmatchedProviderVariantBreak reports a response-side union failure: the
// provider may produce a variant that no consumer variant accepts. One break
// stands in for the whole failed variant; per-candidate sub-breaks are never
// surfaced.
func NewUnmatchedProviderVariantBreak(
	checkedResource *model.PersistedResource,
	counterpartResource *model.PersistedResource,
	providerVariantPath string,
) ContractBreakingChange {
	consumerResource, providerResource := checkedResource, counterpartResource
	if !checkedResource.IsConsumer() {
		consumerResource, providerResource = counterpartResource, checkedResource
	}

	return ContractBreakingChange{
		CheckedResource:     checkedResource,
		CounterpartResource: counterpartResource,
		Reason:              ReasonPropertyNotMatchingAnyVariant,
		Details: map[string]string{
			"property":             providerVariantPath,
			"consumerPropertyType": typeToken(consumerResource, anyOfNodePath(providerVariantPath)),
			"providerPropertyType": typeToken(providerResource, providerVariantPath),
		},
	}
}

// NewUnmatchedConsumerVariantBreak reports a request-side union failure: the
// consumer may send a variant that no provider variant accepts. One break
// stands in for the whole failed variant; per-candidate sub-breaks are never
// surfaced.
func NewUnmatchedConsumerVariantBreak(
	checkedResource *model.PersistedResource,
	counterpartResource *model.PersistedResource,
	consumerVariantPath string,
) ContractBreakingChange {
	consumerResource, providerResource := checkedResource, counterpartResource
	if !checkedResource.IsConsumer() {
		consumerResource, providerResource = counterpartResource, checkedResource
	}

	return ContractBreakingChange{
		CheckedResource:     checkedResource,
		CounterpartResource: counterpartResource,
		Reason:              ReasonPropertyNotMatchingAnyVariant,
		Details: map[string]string{
			"property":             consumerVariantPath,
			"consumerPropertyType": typeToken(consumerResource, consumerVariantPath),
			"providerPropertyType": typeToken(providerResource, anyOfNodePath(consumerVariantPath)),
		},
	}
}

// anyOfNodePath trims a trailing #N variant index; the provider side renders
// its type at the anyOf node itself, not at the consumer's variant index.
func anyOfNodePath(variantPath string) string {
	hash := strings.LastIndex(variantPath, "#")
	if hash < 0 || hash+1 == len(variantPath) {
		return variantPath
	}

	if _, err := strconv.Atoi(variantPath[hash+1:]); err != nil {
		return variantPath
	}

	return variantPath[:hash]
}

// typeToken renders a property's type, resolving array item types from the
// flat property map ($.list (array) + $.list[] (string) → "array<string>")
// and anyOf variants from their indexed paths ($.prop (anyOf) + $.prop#0
// (string) + $.prop#1 (number) → "anyOf<string, number>").
func typeToken(resource *model.PersistedResource, propertyPath string) string {
	if resource.Properties[propertyPath].Type == "array" {
		return "array<" + typeToken(resource, propertyPath+"[]") + ">"
	}

	if resource.Properties[propertyPath].Type == "anyOf" {
		var variants []string
		for index := 0; ; index++ {
			variantPath := propertyPath + "#" + strconv.Itoa(index)
			if _, exists := resource.Properties[variantPath]; !exists {
				break
			}
			variants = append(variants, typeToken(resource, variantPath))
		}

		return "anyOf<" + strings.Join(variants, ", ") + ">"
	}

	return resource.Properties[propertyPath].Type
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
