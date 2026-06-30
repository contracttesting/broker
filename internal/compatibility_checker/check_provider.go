package compatibility_checker

import (
	"context"

	"github.com/contracttesting/broker/internal/model"
)

func (c *CompatibilityChecker) checkProvider(
	ctx context.Context,
	providerResource model.CounterpartResource,
	environment *model.Environment,
	report *CompatibilityReport,
) {
	consumers := c.repository.GetConsumersResourcesByProviderResourceAndEnvironment(ctx, providerResource, environment)

	for _, consumerResource := range consumers {
		consumerBreaks := checkResources(&providerResource, &consumerResource)
		for _, breakingChange := range consumerBreaks {
			report.AppendContractBreakChange(breakingChange)
		}

		report.AppendResult(consumerResource.ParticipantName, CompatibilityResult{
			CounterpartParticipantID: consumerResource.ParticipantID,
			CounterpartVersion:       consumerResource.ParticipantVersion.String,
			Deployable:               len(consumerBreaks) == 0,
		})
	}
}
