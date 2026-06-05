package rename_participant

import (
	"github.com/contracttesting/broker/internal/repository"
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

	found, conflict := h.participantRepository.Rename(ctx.Context(), requestBody.OldName, requestBody.NewName)
	if conflict {
		return h.respondAlreadyExists(ctx)
	}

	if !found {
		return h.respondNotFound(ctx)
	}

	return ctx.Status(fiber.StatusOK).JSON(RenameParticipantResponseBody{
		Message: ParticipantRenamed,
	})
}

func (h *RenameParticipantHandler) respondInvalidInput(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusBadRequest).JSON(RenameParticipantResponseBody{
		Message: ParticipantInvalidInput,
	})
}

func (h *RenameParticipantHandler) respondAlreadyExists(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusBadRequest).JSON(RenameParticipantResponseBody{
		Message: ParticipantAlreadyExists,
	})
}

func (h *RenameParticipantHandler) respondNotFound(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusNotFound).JSON(RenameParticipantResponseBody{
		Message: ParticipantNotFound,
	})
}
