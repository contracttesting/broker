package create_participant

import (
	"github.com/contracttesting/broker/internal/model"
	"github.com/contracttesting/broker/internal/repository"
	"github.com/contracttesting/broker/internal/shared"
	"github.com/gofiber/fiber/v3"
)

type CreateParticipantHandler struct {
	participantRepository *repository.ParticipantRepository
}

func NewCreateParticipantHandler(
	participantRepository *repository.ParticipantRepository,
) *CreateParticipantHandler {
	return &CreateParticipantHandler{
		participantRepository: participantRepository,
	}
}

func (ctr *CreateParticipantHandler) Handle(ctx fiber.Ctx) error {
	requestBody := &CreateParticipantRequestBody{}
	if err := ctx.Bind().JSON(requestBody); err != nil {
		return ctr.respondInvalidInput(ctx)
	}

	if requestBody.Participant == "" {
		return ctr.respondInvalidInput(ctx)
	}

	if ctr.participantRepository.ExistsByName(ctx.Context(), requestBody.Participant) {
		return ctr.respondAlreadyExists(ctx)
	}

	ctr.participantRepository.Create(ctx.Context(), model.NewParticipant(requestBody.Participant))

	return ctx.Status(fiber.StatusOK).JSON(CreateParticipantResponseBody{
		BrokerResponseBody: shared.BrokerResponseBody{
			Success: true,
			Message: ParticipantCreated,
		},
	})
}

func (ctr *CreateParticipantHandler) respondInvalidInput(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusBadRequest).JSON(CreateParticipantResponseBody{
		BrokerResponseBody: shared.BrokerResponseBody{
			Success: false,
			Message: ParticipantInvalidInput,
		},
	})
}

func (ctr *CreateParticipantHandler) respondAlreadyExists(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusOK).JSON(CreateParticipantResponseBody{
		BrokerResponseBody: shared.BrokerResponseBody{
			Success: true,
			Message: ParticipantAlreadyExists,
		},
	})
}
