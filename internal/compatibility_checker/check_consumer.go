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
	consumerResource model.PersistedResource,
	environment *model.Environment,
	report *ContractCompatibilityReport,
) {
	providerResource, err := c.repository.GetProviderResourceByConsumerResource(ctx, consumerResource.ProviderHash)

	incompatibleItem := NewIncompatibleItem()

	if errors.Is(err, repository.ErrProviderResourceNotFound) {
		incompatibleItem.AppendContractBreakChange(
			NewProviderResourceNotFound(&consumerResource),
		)

		report.AppendResult(consumerResource.ConsumedProvider.String, incompatibleItem)

		return
	}

	version, deployed := providerResource.DeployedVersionIn(environment.Name)
	if !deployed {
		incompatibleItem.IncompatibleCounterpart.ParticipantID = providerResource.ParticipantID

		incompatibleItem.AppendContractBreakChange(NewProviderResourceNotDeployed(
			&consumerResource,
			providerResource.DeployedEnvironments(),
		))

		report.AppendResult(consumerResource.ConsumedProvider.String, incompatibleItem)

		return
	}

	providerResource.ParticipantVersion = null.StringFrom(version)
	incompatibleItem.IncompatibleCounterpart.ParticipantID = providerResource.ParticipantID
	incompatibleItem.IncompatibleCounterpart.ParticipantVersion = providerResource.ParticipantVersion
	incompatibleItem.IncompatibleCounterpart.ParticipantName = providerResource.ParticipantName

	for _, breakingChange := range checkResources(&consumerResource, &providerResource) {
		incompatibleItem.AppendContractBreakChange(breakingChange)
	}

	report.AppendResult(consumerResource.ConsumedProvider.String, incompatibleItem)
}
