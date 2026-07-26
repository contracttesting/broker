package compatibility_checker

import (
	"github.com/contracttesting/broker/internal/model"
)

type propertyBreak struct {
	reason BreakingReason
	path   string
}

func compareResponseProperties(consumer, provider map[string]model.Property) []propertyBreak {
	var breaks []propertyBreak

	for path, consumerProperty := range consumer {
		providerProperty, propertyExists := provider[path]

		// If the property is not present in the provider and is required in the consumer, it is a breaking change.
		if !propertyExists {
			if !consumerProperty.Optional {
				breaks = append(breaks, propertyBreak{ReasonPropertyMissingInProvider, path})
			}

			continue
		}

		// If the property is present in the provider and the type is different, it is a breaking change.
		if consumerProperty.Type != providerProperty.Type {
			breaks = append(breaks, propertyBreak{ReasonPropertyTypeMismatch, path})

			continue
		}

		// If the property is required in the consumer and is optional in the provider, it is a breaking change.
		if !consumerProperty.Optional && providerProperty.Optional {
			breaks = append(breaks, propertyBreak{ReasonPropertyOptionalInProviderRequiredInConsumer, path})
		}
	}

	return breaks
}

func compareRequestProperties(consumer, provider map[string]model.Property) []propertyBreak {
	var breaks []propertyBreak

	for path, providerProperty := range provider {
		consumerProperty, propertyExists := consumer[path]

		// If the property is not present in the consumer and is not optional, it is a breaking change.
		if !propertyExists {
			if !providerProperty.Optional {
				breaks = append(breaks, propertyBreak{ReasonPropertyMissingInConsumer, path})
			}

			continue
		}

		// If the property is present in the consumer and the type is different, it is a breaking change.
		if consumerProperty.Type != providerProperty.Type {
			breaks = append(breaks, propertyBreak{ReasonPropertyTypeMismatch, path})

			continue
		}

		// If the property is required in the provider and is optional in the consumer, it is a breaking change.
		if !providerProperty.Optional && consumerProperty.Optional {
			breaks = append(breaks, propertyBreak{ReasonPropertyOptionalInConsumerRequiredInProvider, path})
		}
	}

	return breaks
}
