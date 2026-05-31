package create_participant

import (
	"github.com/contracttesting/broker/internal/model"
	"github.com/contracttesting/broker/internal/repository"
	"github.com/gofiber/fiber/v3"
)

type createParticipantRequest struct {
	Name string `json:"name"`
}

type CreateParticipantHandler struct {
	participantRepository *repository.ParticipantRepository
}

func NewCreateParticipantHandler(repo *repository.ParticipantRepository) *CreateParticipantHandler {
	return &CreateParticipantHandler{participantRepository: repo}
}

func (ctr *CreateParticipantHandler) Handle(ctx fiber.Ctx) error {
	request := &createParticipantRequest{}

	if err := ctx.Bind().JSON(request); err != nil {
		return ctr.respondInvalidInput(ctx)
	}

	if request.Name == "" {
		return ctr.respondInvalidInput(ctx)
	}

	if ctr.participantRepository.ExistsByName(ctx.Context(), request.Name) {
		return ctr.respondAlreadyExists(ctx)
	}

	ctr.participantRepository.Create(ctx.Context(), model.NewParticipant(request.Name))

	return ctx.Status(fiber.StatusOK).JSON(CreateParticipantResponse{
		Success: true,
		Message: string(ParticipantCreated),
	})
}

func (ctr *CreateParticipantHandler) respondInvalidInput(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusBadRequest).JSON(CreateParticipantResponse{
		Success: false,
		Message: ParticipantInvalidInput,
	})
}

func (ctr *CreateParticipantHandler) respondAlreadyExists(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusOK).JSON(CreateParticipantResponse{
		Success: true,
		Message: ParticipantAlreadyExists,
	})
}
