package compatibility_checker

import (
	"context"
	"errors"

	"github.com/contracttesting/broker/internal/model"
	"github.com/contracttesting/broker/internal/repository"
	"github.com/guregu/null"
)

func (c *CompatibilityChecker) checkConsumer(
	ctx context.Context,
	consumerResource model.CounterpartResource,
	environment *model.Environment,
	report *CompatibilityReport,
) {
	providerResource, err := c.repository.GetProviderResourceByConsumerResource(ctx, consumerResource)

	if errors.Is(err, repository.ErrProviderResourceNotFound) {
		report.AppendContractBreakChange(NewContractBreakingChange(
			&consumerResource,
			nil,
			ReasonProviderResourceNotFound,
		))

		report.AppendResult(consumerResource.ConsumedProvider.String, CompatibilityResult{
			Deployable: false,
		})

		return
	}

	version, deployed := providerResource.DeployedVersionIn(environment.Name)
	if !deployed {
		report.AppendContractBreakChange(NewProviderNotDeployedBreakingChange(
			&consumerResource,
			providerResource.DeployedEnvironments(),
		))

		report.AppendResult(consumerResource.ConsumedProvider.String, CompatibilityResult{
			Deployable: false,
		})

		return
	}

	providerResource.ParticipantVersion = null.StringFrom(version)

	breaks := checkResources(&consumerResource, &providerResource)
	for _, breakingChange := range breaks {
		report.AppendContractBreakChange(breakingChange)
	}

	report.AppendResult(consumerResource.ConsumedProvider.String, CompatibilityResult{
		CounterpartParticipantID: providerResource.ParticipantID,
		CounterpartVersion:       providerResource.ParticipantVersion.String,
		Deployable:               len(breaks) == 0,
	})
}
