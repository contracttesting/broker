package compatibility_checker

import "github.com/contracttesting/broker/internal/model"

func checkRequestResource(
	checked *model.Resource,
	counterpart *model.Resource,
	consumer *model.Resource,
	provider *model.Resource,
) []BreakingChange {
	var breaks []BreakingChange

	for providerPropertyPath, providerProperty := range provider.Properties {
		consumerProperty, propertyExists := consumer.Properties[providerPropertyPath]
		// If the property is not present in the consumer and is not optional, it is a breaking change.
		if !propertyExists && !providerProperty.Optional {
			breaks = append(breaks, NewPropertyBreakChange(
				checked,
				counterpart,
				ReasonPropertyMissingInConsumer,
				providerPropertyPath,
			))

			continue
		}

		// If the property is present in the consumer and the type is different, it is a breaking change.
		if consumerProperty.Type != providerProperty.Type {
			breaks = append(breaks, NewPropertyBreakChange(
				checked,
				counterpart,
				ReasonPropertyTypeMismatch,
				providerPropertyPath,
			))

			continue
		}

		// If the property is required in the provider and is optional in the consumer, it is a breaking change.
		if !providerProperty.Optional && consumerProperty.Optional {
			breaks = append(breaks, NewPropertyBreakChange(
				checked,
				counterpart,
				ReasonPropertyOptionalInConsumerRequiredInProvider,
				providerPropertyPath,
			))
		}
	}

	return breaks
}
