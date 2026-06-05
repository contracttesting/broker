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

	contract, exists := h.contractRepository.LoadContractByNameAndVersion(
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

	report := h.compatibilityChecker.Check(
		ctx.Context(),
		contract,
		environment,
	)

	for _, result := range report.Results {
		h.compatibilityMatrixRepository.Insert(ctx.Context(), &model.CompatibilityMatrix{
			ParticipantID:            contract.ParticipantID(),
			Version:                  contract.Version,
			CounterpartParticipantID: result.CounterpartParticipantID,
			CounterpartVersion:       result.CounterpartVersion,
			Deployable:               result.Deployable,
		})
	}

	return ctx.Status(fiber.StatusOK).JSON(CanIDeployResponseBody{
		Message:    "Contract checked successfully",
		Deployable: len(report.Breaks) == 0,
		Breaks:     report.Breaks,
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
