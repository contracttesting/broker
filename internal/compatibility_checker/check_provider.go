package compatibility_checker

import (
	"context"

	"github.com/contracttesting/broker/internal/model"
)

func (c *CompatibilityChecker) checkProvider(
	ctx context.Context,
	providerResource model.PersistedResource,
	environment *model.Environment,
	report *ContractCompatibilityReport,
) {
	consumers := c.repository.GetConsumersResourcesByProviderHashAndEnvironmentID(
		ctx,
		providerResource.ProviderHash,
		environment.ID,
	)

	for _, consumerResource := range consumers {
		result := NewIncompatibleItem()
		result.IncompatibleCounterpart.ParticipantID = consumerResource.ParticipantID
		result.IncompatibleCounterpart.ParticipantVersion = consumerResource.ParticipantVersion
		result.IncompatibleCounterpart.ParticipantName = consumerResource.ParticipantName

		for _, breakingChange := range CheckResources(&providerResource, &consumerResource) {
			result.AppendContractBreakChange(breakingChange)
		}

		report.AppendResult(consumerResource.ParticipantName, result)
	}
}
