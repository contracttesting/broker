package can_i_deploy

import (
	"github.com/contracttesting/broker/internal/compatibility_checker"
	"github.com/contracttesting/broker/internal/model"
	"github.com/contracttesting/broker/internal/repository"
	"github.com/gofiber/fiber/v3"
	"github.com/guregu/null"
)

type CanIDeployHandler struct {
	contractRepository           *repository.ContractRepository
	environmentRepository        *repository.EnvironmentRepository
	compatibilityCheckRepository *repository.CompatibilityCheckRepository
	compatibilityChecker         *compatibility_checker.CompatibilityChecker
	participantRepository        *repository.ParticipantRepository
}

func NewCanIDeployHandler(
	contractRepository *repository.ContractRepository,
	environmentRepository *repository.EnvironmentRepository,
	compatibilityCheckRepository *repository.CompatibilityCheckRepository,
	compatibilityChecker *compatibility_checker.CompatibilityChecker,
	participantRepository *repository.ParticipantRepository,
) *CanIDeployHandler {
	return &CanIDeployHandler{
		contractRepository:           contractRepository,
		environmentRepository:        environmentRepository,
		compatibilityCheckRepository: compatibilityCheckRepository,
		compatibilityChecker:         compatibilityChecker,
		participantRepository:        participantRepository,
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

	compatibilityReport := h.compatibilityChecker.Check(
		ctx.Context(),
		contract,
		environment,
	)

	check := &model.CompatibilityCheck{
		ParticipantID: contract.ParticipantID,
		Version:       contract.Version,
		EnvironmentID: environment.ID,
		Deployable:    true,
	}

	for counterpartName, result := range compatibilityReport.Results {
		check.Deployable = check.Deployable && result.Deployable

		counterpartID := result.IncompatibleCounterpart.ParticipantID
		if counterpartID == 0 {
			// the counterpart never published the consumed resource; its
			// participant row may still exist and keep the verdict addressable
			counterpart, exists := h.participantRepository.FindByName(ctx.Context(), counterpartName)
			if !exists {
				continue
			}
			counterpartID = counterpart.ID
		}

		check.Results = append(check.Results, model.CompatibilityCheckResult{
			CounterpartParticipantID: counterpartID,
			CounterpartVersion:       result.IncompatibleCounterpart.ParticipantVersion,
			Deployable:               result.Deployable,
			Reason:                   representativeReason(result),
		})
	}

	h.compatibilityCheckRepository.Insert(ctx.Context(), check)

	return ctx.Status(fiber.StatusOK).JSON(CanIDeployResponseBody{
		Message:     "Contract checked successfully",
		Participant: requestBody.Participant,
		Version:     requestBody.Version,
		Environment: requestBody.Environment,
		Deployable:  check.Deployable,
		Results:     compatibilityReport.Hierarchical,
	})
}

func representativeReason(result compatibility_checker.IncompatibleItem) null.String {
	if result.Deployable {
		return null.String{}
	}

	return null.StringFrom(string(result.Breaks[0].Reason))
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
