package can_i_deploy

import (
	"context"

	"github.com/contracttesting/broker/internal/compatibility_checker"
	"github.com/contracttesting/broker/internal/model"
	"github.com/contracttesting/broker/internal/repository"
	"github.com/gofiber/fiber/v3"
	"github.com/guregu/null"
)

type CanIDeployHandler struct {
	contractRepository      *repository.ContractRepository
	environmentRepository   *repository.EnvironmentRepository
	compatibilityRepository *repository.CompatibilityRepository
	compatibilityChecker    *compatibility_checker.CompatibilityChecker
	participantRepository   *repository.ParticipantRepository
}

func NewCanIDeployHandler(
	contractRepository *repository.ContractRepository,
	environmentRepository *repository.EnvironmentRepository,
	compatibilityRepository *repository.CompatibilityRepository,
	compatibilityChecker *compatibility_checker.CompatibilityChecker,
	participantRepository *repository.ParticipantRepository,
) *CanIDeployHandler {
	return &CanIDeployHandler{
		contractRepository:      contractRepository,
		environmentRepository:   environmentRepository,
		compatibilityRepository: compatibilityRepository,
		compatibilityChecker:    compatibilityChecker,
		participantRepository:   participantRepository,
	}
}

func (h *CanIDeployHandler) Handle(ctx fiber.Ctx) error {
	requestBody := &CanIDeployRequestBody{}
	if err := ctx.Bind().JSON(requestBody); err != nil {
		return h.respondInvalidInput(ctx)
	}

	if requestBody.Participant == "" || requestBody.Version == "" || requestBody.Environment == "" {
		return h.respondInvalidInput(ctx)
	}

	participant, exists := h.participantRepository.FindByName(ctx.Context(), requestBody.Participant)
	if !exists {
		return h.respondParticipantNotFound(ctx)
	}

	contract, exists := h.contractRepository.GetContractByNameAndVersion(
		ctx.Context(),
		participant.Name,
		requestBody.Version,
	)

	if !exists {
		return h.respondContractNotFound(ctx)
	}

	environment, exists := h.environmentRepository.FindByName(ctx.Context(), requestBody.Environment)
	if !exists {
		return h.respondInvalidInput(ctx)
	}

	counterparts := h.contractRepository.LoadCounterparts(ctx.Context(), contract, environment.ID)

	compatibilityReport := h.compatibilityChecker.Check(
		ctx.Context(),
		contract,
		environment,
		counterparts,
	)

	deployable := true
	for _, result := range compatibilityReport.Results {
		deployable = deployable && result.Deployable
	}

	h.recordCheck(ctx.Context(), contract, environment, deployable, compatibilityReport)

	return ctx.Status(fiber.StatusOK).JSON(CanIDeployResponseBody{
		Message:     "Contract checked successfully",
		Participant: requestBody.Participant,
		Version:     requestBody.Version,
		Environment: requestBody.Environment,
		Deployable:  deployable,
		Results:     compatibilityReport.Hierarchical,
	})
}

func (h *CanIDeployHandler) recordCheck(
	ctx context.Context,
	contract *model.PersistedContract,
	environment *model.Environment,
	deployable bool,
	report *compatibility_checker.ContractCompatibilityReport,
) {
	check := &model.CompatibilityCheck{
		ParticipantID: contract.ParticipantID,
		ContractID:    contract.ID,
		Version:       contract.Version,
		EnvironmentID: environment.ID,
		Deployable:    deployable,
	}

	results := make([]model.CompatibilityCheckResult, 0, len(report.Results))
	breaksByPair := make(map[[2]int64][]model.VerdictBreak)

	for counterpartName, item := range report.Results {
		counterpart := item.IncompatibleCounterpart

		result := model.CompatibilityCheckResult{
			CounterpartName:    counterpartName,
			CounterpartVersion: counterpart.ParticipantVersion,
			Deployable:         item.Deployable,
		}

		if counterpart.ParticipantID > 0 {
			result.CounterpartParticipantID = null.IntFrom(counterpart.ParticipantID)
		}

		if counterpart.ContractID > 0 {
			contractIDOne, contractIDTwo := model.OrderContractPair(contract.ID, counterpart.ContractID)
			pair := [2]int64{contractIDOne, contractIDTwo}

			result.VerdictContractIDOne = null.IntFrom(contractIDOne)
			result.VerdictContractIDTwo = null.IntFrom(contractIDTwo)

			// A cached verdict is already stored: only the pairs compared in this call become facts.
			if !item.VerdictCached {
				breaks, exists := breaksByPair[pair]
				if !exists {
					breaks = make([]model.VerdictBreak, 0, len(item.Breaks))
				}

				for _, breakChange := range item.Breaks {
					if !breakChange.IsPropertyBreak() {
						continue
					}

					breaks = append(breaks, model.VerdictBreak{
						Endpoint:    breakChange.CheckedResource.Endpoint,
						Method:      breakChange.CheckedResource.Method,
						Interaction: breakChange.InteractionKey(),
						Reason:      string(breakChange.Reason),
						Details:     breakChange.Details,
					})
				}

				breaksByPair[pair] = breaks
			}
		}

		results = append(results, result)
	}

	newVerdicts := make([]model.CompatibilityVerdict, 0, len(breaksByPair))
	for pair, breaks := range breaksByPair {
		newVerdicts = append(newVerdicts, model.CompatibilityVerdict{
			ContractIDOne: pair[0],
			ContractIDTwo: pair[1],
			Breaks:        breaks,
		})
	}

	h.compatibilityRepository.RecordCheck(ctx, check, results, newVerdicts)
}

func (h *CanIDeployHandler) respondParticipantNotFound(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusNotFound).JSON(CanIDeployErrorResponseBody{
		Message: ParticipantNotFound,
	})
}

func (h *CanIDeployHandler) respondInvalidInput(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusBadRequest).JSON(CanIDeployErrorResponseBody{
		Message: "Invalid input",
	})
}

func (h *CanIDeployHandler) respondContractNotFound(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusNotFound).JSON(CanIDeployErrorResponseBody{
		Message: ContractNotFound,
	})
}
