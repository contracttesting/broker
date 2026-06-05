package compatibility_checker

import "github.com/contracttesting/broker/internal/model"

func checkResponseResource(
	checked *model.Resource,
	counterpart *model.Resource,
	consumer *model.Resource,
	provider *model.Resource,
) []BreakingChange {
	var breaks []BreakingChange

	for consumerPropertyPath, consumerProperty := range consumer.Properties {
		providerProperty, propertyExists := provider.Properties[consumerPropertyPath]

		// If the property is not present in the provider and is required in the consumer, it is a breaking change.
		if !propertyExists && !consumerProperty.Optional {
			breaks = append(breaks, NewPropertyBreakChange(
				checked,
				counterpart,
				ReasonMissingInProvider,
				consumerPropertyPath,
			))

			continue
		}

		// If the property is present in the provider and the type is different, it is a breaking change.
		if consumerProperty.Type != providerProperty.Type {
			breaks = append(breaks, NewPropertyBreakChange(
				checked,
				counterpart,
				ReasonTypeMismatch,
				consumerPropertyPath,
			))

			continue
		}

		// If the property is required in the consumer and is optional in the provider, it is a breaking change.
		if !consumerProperty.Optional && providerProperty.Optional {
			breaks = append(breaks, NewPropertyBreakChange(
				checked,
				counterpart,
				ReasonOptionalInProviderRequiredInConsumer,
				consumerPropertyPath,
			))
		}
	}

	return breaks
}
