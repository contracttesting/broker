package compatibility_checker

import "github.com/contracttesting/broker/internal/model"

func checkResources(checked *model.Resource, counterpart *model.Resource) []BreakingChange {
	consumer, provider := checked, counterpart
	if !checked.IsConsumer() {
		consumer, provider = counterpart, checked
	}

	switch consumer.Interaction {
	case model.RestRequest:
		return checkRequestResource(checked, counterpart, consumer, provider)
	default:
		return checkResponseResource(checked, counterpart, consumer, provider)
	}
}
