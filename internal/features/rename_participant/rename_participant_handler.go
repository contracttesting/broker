package rename_participant

import (
	"github.com/contracttesting/broker/internal/repository"
	"github.com/contracttesting/broker/internal/shared"
	"github.com/gofiber/fiber/v3"
)

type RenameParticipantHandler struct {
	participantRepository *repository.ParticipantRepository
}

func NewRenameParticipantHandler(repo *repository.ParticipantRepository) *RenameParticipantHandler {
	return &RenameParticipantHandler{participantRepository: repo}
}

func (h *RenameParticipantHandler) Handle(ctx fiber.Ctx) error {
	requestBody := &RenameParticipantRequestBody{}
	if err := ctx.Bind().JSON(requestBody); err != nil {
		return h.respondInvalidInput(ctx)
	}

	if requestBody.OldName == "" || requestBody.NewName == "" {
		return h.respondInvalidInput(ctx)
	}

	if _, conflict := h.participantRepository.Rename(ctx.Context(), requestBody.OldName, requestBody.NewName); conflict {
		return h.respondAlreadyExists(ctx)
	}

	return ctx.Status(fiber.StatusOK).JSON(RenameParticipantResponseBody{
		BrokerResponseBody: shared.BrokerResponseBody{
			Success: true,
			Message: ParticipantRenamed,
		},
	})
}

func (h *RenameParticipantHandler) respondInvalidInput(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusBadRequest).JSON(RenameParticipantResponseBody{
		BrokerResponseBody: shared.BrokerResponseBody{
			Success: false,
			Message: ParticipantInvalidInput,
		},
	})
}

func (h *RenameParticipantHandler) respondAlreadyExists(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusBadRequest).JSON(RenameParticipantResponseBody{
		BrokerResponseBody: shared.BrokerResponseBody{
			Success: false,
			Message: ParticipantAlreadyExists,
		},
	})
}
