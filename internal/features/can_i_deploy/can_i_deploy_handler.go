package can_i_deploy

import (
	"github.com/contracttesting/broker/internal/compatibility_checker"
	"github.com/contracttesting/broker/internal/model"
	"github.com/contracttesting/broker/internal/repository"
	"github.com/gofiber/fiber/v3"
)

type CanIDeployHandler struct {
	contractRepository            *repository.ContractRepository
	environmentRepository         *repository.EnvironmentRepository
	compatibilityMatrixRepository *repository.CompatibilityMatrixRepository
	compatibilityChecker          *compatibility_checker.CompatibilityChecker
	participantRepository         *repository.ParticipantRepository
}

func NewCanIDeployHandler(
	contractRepository *repository.ContractRepository,
	environmentRepository *repository.EnvironmentRepository,
	compatibilityMatrixRepository *repository.CompatibilityMatrixRepository,
	compatibilityChecker *compatibility_checker.CompatibilityChecker,
	participantRepository *repository.ParticipantRepository,
) *CanIDeployHandler {
	return &CanIDeployHandler{
		contractRepository:            contractRepository,
		environmentRepository:         environmentRepository,
		compatibilityMatrixRepository: compatibilityMatrixRepository,
		compatibilityChecker:          compatibilityChecker,
		participantRepository:         participantRepository,
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

	deployable := true
	for _, result := range compatibilityReport.Results {
		deployable = deployable && result.Deployable

		h.compatibilityMatrixRepository.Insert(ctx.Context(), &model.CompatibilityMatrix{
			ParticipantID:            contract.ParticipantID,
			Version:                  contract.Version,
			CounterpartParticipantID: result.IncompatibleCounterpart.ParticipantID,
			CounterpartVersion:       result.IncompatibleCounterpart.ParticipantVersion.ValueOrZero(),
			Deployable:               result.Deployable,
		})
	}

	return ctx.Status(fiber.StatusOK).JSON(CanIDeployResponseBody{
		Message:     "Contract checked successfully",
		Participant: requestBody.Participant,
		Version:     requestBody.Version,
		Environment: requestBody.Environment,
		Deployable:  deployable,
		Results:     compatibilityReport.Results,
	})
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
