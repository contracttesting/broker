package record_deployment

import (
	"github.com/contracttesting/broker/internal/model"
	"github.com/contracttesting/broker/internal/repository"
	"github.com/contracttesting/broker/internal/shared"
	"github.com/gofiber/fiber/v3"
)

type RecordDeploymentHandler struct {
	deploymentRepository  *repository.DeploymentRepository
	participantRepository *repository.ParticipantRepository
	contractRepository    *repository.ContractRepository
	environmentRepository *repository.EnvironmentRepository
}

func NewRecordDeploymentHandler(
	deploymentRepository *repository.DeploymentRepository,
	participantRepository *repository.ParticipantRepository,
	contractRepository *repository.ContractRepository,
	environmentRepository *repository.EnvironmentRepository,
) *RecordDeploymentHandler {
	return &RecordDeploymentHandler{
		deploymentRepository:  deploymentRepository,
		participantRepository: participantRepository,
		contractRepository:    contractRepository,
		environmentRepository: environmentRepository,
	}
}

func (h *RecordDeploymentHandler) Handle(ctx fiber.Ctx) error {
	requestBody := &RecordDeploymentRequestBody{}
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

	environment, exists := h.environmentRepository.FindByName(ctx.Context(), requestBody.Environment)
	if !exists {
		return h.respondEnvironmentNotFound(ctx)
	}

	h.deploymentRepository.Insert(ctx.Context(), model.NewDeployment(participant, requestBody.Version, environment))

	return ctx.Status(fiber.StatusOK).JSON(RecordDeploymentResponseBody{
		BrokerResponseBody: shared.BrokerResponseBody{
			Success: true,
			Message: DeploymentRecorded,
		},
	})
}

func (h *RecordDeploymentHandler) respondInvalidInput(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusBadRequest).JSON(RecordDeploymentResponseBody{
		BrokerResponseBody: shared.BrokerResponseBody{
			Success: false,
			Message: DeploymentInvalidInput,
		},
	})
}

func (h *RecordDeploymentHandler) respondEnvironmentNotFound(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusNotFound).JSON(RecordDeploymentResponseBody{
		BrokerResponseBody: shared.BrokerResponseBody{
			Success: false,
			Message: EnvironmentNotFound,
		},
	})
}

func (h *RecordDeploymentHandler) respondParticipantNotFound(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusNotFound).JSON(RecordDeploymentResponseBody{
		BrokerResponseBody: shared.BrokerResponseBody{
			Success: false,
			Message: ParticipantNotFound,
		},
	})
}
