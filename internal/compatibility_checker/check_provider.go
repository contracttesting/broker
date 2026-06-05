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

		report.Results = append(report.Results, CompatibilityResult{
			CounterpartParticipantID: consumer.ParticipantID(),
			CounterpartVersion:       consumer.Version,
			Deployable:               len(consumerBreaks) == 0,
		})
	}
}
