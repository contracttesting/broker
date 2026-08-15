package compatibility_checker

import (
	"context"

	"github.com/contracttesting/broker/internal/model"
)

// checkRemovedProvider runs outside the pair cache: the pair verdict may have been stored by
// the consumer's own check, which never sees the removal, so it is always resolved live.
func (c *CompatibilityChecker) checkRemovedProvider(
	ctx context.Context,
	removedResource model.PersistedResource,
	environment *model.Environment,
	report *ContractCompatibilityReport,
) {
	consumers := c.repository.GetConsumersResourcesByProviderHashAndEnvironmentID(
		ctx,
		removedResource.ProviderHash,
		environment.ID,
	)

	for _, result := range CheckRemovedProviderResource(&removedResource, consumers) {
		report.AppendResult(result.IncompatibleCounterpart.ParticipantName, result)
	}
}

func CheckRemovedProviderResource(
	removedResource *model.PersistedResource,
	consumers []model.PersistedResource,
) []*IncompatibleItem {
	results := make([]*IncompatibleItem, 0, len(consumers))

	for _, consumerResource := range consumers {
		// ContractID is left zeroed so this break alone never persists a verdict for the pair.
		result := NewIncompatibleItem()
		result.IncompatibleCounterpart.ParticipantID = consumerResource.ParticipantID
		result.IncompatibleCounterpart.ParticipantName = consumerResource.ParticipantName
		result.IncompatibleCounterpart.ParticipantVersion = consumerResource.ParticipantVersion

		result.AppendContractBreakChange(
			NewProviderResourceRemovedButStillConsumed(removedResource, &consumerResource),
		)

		results = append(results, result)
	}

	return results
}
