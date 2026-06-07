package compatibility_checker

import (
	"context"
	"errors"

	"github.com/contracttesting/broker/internal/model"
	"github.com/contracttesting/broker/internal/repository"
)

func (c *CompatibilityChecker) checkConsumer(
	ctx context.Context,
	consumer model.Resource,
	environment *model.Environment,
	report *CompatibilityReport,
) {
	provider, err := c.repository.LoadProviderResourceOfConsumer(ctx, consumer)

	if errors.Is(err, repository.ErrProviderResourceNotFound) {
		report.Append(NewBreakingChange(
			&consumer,
			nil,
			ReasonProviderResourceNotFound,
		))

		report.AppendResult(consumer.ConsumedProvider, CompatibilityResult{
			Deployable: false,
		})

		return
	}

	version, deployed := provider.DeployedVersionIn(environment.Name)
	if !deployed {
		report.Append(NewProviderNotDeployedBreakingChange(
			&consumer,
			provider.DeployedEnvironments(),
		))

		report.AppendResult(consumer.ConsumedProvider, CompatibilityResult{
			Deployable: false,
		})

		return
	}

	provider.Version = version

	breaks := checkResources(&consumer, &provider)
	for _, breakingChange := range breaks {
		report.Append(breakingChange)
	}

	report.AppendResult(consumer.ConsumedProvider, CompatibilityResult{
		CounterpartParticipantID: provider.ParticipantID(),
		CounterpartVersion:       provider.Version,
		Deployable:               len(breaks) == 0,
	})
}
