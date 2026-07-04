package compatibility_checker

import (
	"context"

	"github.com/contracttesting/broker/internal/model"
	"github.com/contracttesting/broker/internal/repository"
)

type CompatibilityChecker struct {
	repository *repository.ContractRepository
}

func NewCompatibilityChecker(repository *repository.ContractRepository) *CompatibilityChecker {
	return &CompatibilityChecker{
		repository: repository,
	}
}

func (c *CompatibilityChecker) Check(
	ctx context.Context,
	contract *model.PersistedContract,
	environment *model.Environment,
) *ContractCompatibilityReport {
	report := NewContractCompatibilityReport(
		contract.ParticipantName,
		contract.Version,
		environment.Name,
	)

	for _, resource := range contract.Resources {
		switch resource.Direction {
		case model.Consumes:
			c.checkConsumer(ctx, resource, environment, report)
		case model.Provides:
			c.checkProvider(ctx, resource, environment, report)
		}
	}

	return report
}
