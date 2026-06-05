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
	uploaded *model.Contract,
	environment *model.Environment,
) *CompatibilityReport {
	report := NewCompatibilityReport()

	for _, resource := range uploaded.Resources {
		switch resource.Direction {
		case model.Consumes:
			c.checkConsumer(ctx, resource, environment, report)
		case model.Provides:
			c.checkProvider(ctx, resource, environment, report)
		}
	}

	return report
}
