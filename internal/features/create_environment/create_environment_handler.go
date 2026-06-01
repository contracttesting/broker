package create_environment

import (
	"github.com/contracttesting/broker/internal/model"
	"github.com/contracttesting/broker/internal/repository"
	"github.com/contracttesting/broker/internal/shared"
	"github.com/gofiber/fiber/v3"
)

type CreateEnvironmentHandler struct {
	environmentRepository *repository.EnvironmentRepository
}

func NewCreateEnvironmentHandler(repo *repository.EnvironmentRepository) *CreateEnvironmentHandler {
	return &CreateEnvironmentHandler{environmentRepository: repo}
}

func (ctr *CreateEnvironmentHandler) Handle(ctx fiber.Ctx) error {
	requestBody := &CreateEnvironmentRequestBody{}

	if err := ctx.Bind().JSON(requestBody); err != nil {
		return ctr.respondInvalidInput(ctx)
	}

	if requestBody.Participant == "" {
		return ctr.respondInvalidInput(ctx)
	}

	if ctr.environmentRepository.ExistsByName(ctx.Context(), requestBody.Participant) {
		return ctr.respondAlreadyExists(ctx)
	}

	ctr.environmentRepository.Create(ctx.Context(), model.NewEnvironment(requestBody.Participant))

	return ctx.Status(fiber.StatusOK).JSON(CreateEnvironmentResponseBody{
		BrokerResponseBody: shared.BrokerResponseBody{
			Success: true,
			Message: EnvironmentCreated,
		},
	})
}

func (ctr *CreateEnvironmentHandler) respondInvalidInput(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusBadRequest).JSON(CreateEnvironmentResponseBody{
		BrokerResponseBody: shared.BrokerResponseBody{
			Success: false,
			Message: EnvironmentInvalidInput,
		},
	})
}

func (ctr *CreateEnvironmentHandler) respondAlreadyExists(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusBadRequest).JSON(CreateEnvironmentResponseBody{
		BrokerResponseBody: shared.BrokerResponseBody{
			Success: false,
			Message: EnvironmentAlreadyExists,
		},
	})
}
