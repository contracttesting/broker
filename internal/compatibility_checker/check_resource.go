package compatibility_checker

import "github.com/contracttesting/broker/internal/model"

func CheckResources(checked *model.PersistedResource, counterpart *model.PersistedResource) []ContractBreakingChange {
	consumer, provider := checked, counterpart
	if !checked.IsConsumer() {
		consumer, provider = counterpart, checked
	}

	compare := compareResponseProperties
	if consumer.Interaction == model.RestRequest {
		compare = compareRequestProperties
	}

	var breaks []ContractBreakingChange
	for _, propertyBreak := range compare(consumer.Properties, provider.Properties) {
		if propertyBreak.reason == ReasonPropertyNotMatchingAnyVariant {
			// the unmatched variant belongs to whichever side produces the
			// value: the consumer for requests, the provider for responses
			if consumer.Interaction == model.RestRequest {
				breaks = append(breaks, NewUnmatchedConsumerVariantBreak(checked, counterpart, propertyBreak.path))
			} else {
				breaks = append(breaks, NewUnmatchedProviderVariantBreak(checked, counterpart, propertyBreak.path))
			}

			continue
		}

		breaks = append(breaks, NewPropertyBreakChange(checked, counterpart, propertyBreak.reason, propertyBreak.path))
	}

	return breaks
}
