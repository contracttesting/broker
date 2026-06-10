package compatibility_checker

import (
	"context"

	"github.com/contracttesting/broker/internal/model"
)

func (c *CompatibilityChecker) checkProvider(
	ctx context.Context,
	provider model.Resource,
	environment *model.Environment,
	report *CompatibilityReport,
) {
	consumers := c.repository.FindConsumersOfProviderAndEnvironment(ctx, provider, environment)

	for _, consumer := range consumers {
		consumerBreaks := checkResources(&provider, &consumer)
		for _, breakingChange := range consumerBreaks {
			report.Append(breakingChange)
		}

		report.AppendResult(consumer.ParticipantName(), CompatibilityResult{
			CounterpartParticipantID: consumer.ParticipantID(),
			CounterpartVersion:       consumer.Version.String,
			Deployable:               len(consumerBreaks) == 0,
		})
	}
}
