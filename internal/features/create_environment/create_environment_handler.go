package create_environment

import (
	"github.com/contracttesting/broker/internal/model"
	"github.com/contracttesting/broker/internal/repository"
	"github.com/gofiber/fiber/v3"
)

type CreateEnvironmentHandler struct {
	environmentRepository *repository.EnvironmentRepository
}

func NewCreateEnvironmentHandler(repo *repository.EnvironmentRepository) *CreateEnvironmentHandler {
	return &CreateEnvironmentHandler{environmentRepository: repo}
}

func (this *CreateEnvironmentHandler) Handle(ctx fiber.Ctx) error {
	requestBody := &CreateEnvironmentRequestBody{}

	if err := ctx.Bind().JSON(requestBody); err != nil {
		return this.respondInvalidInput(ctx)
	}

	if requestBody.Environment == "" {
		return this.respondInvalidInput(ctx)
	}

	if this.environmentRepository.ExistsByName(ctx.Context(), requestBody.Environment) {
		return this.respondAlreadyExists(ctx)
	}

	this.environmentRepository.Create(ctx.Context(), model.NewEnvironment(requestBody.Environment))

	return ctx.Status(fiber.StatusOK).JSON(CreateEnvironmentResponseBody{
		Message: EnvironmentCreated,
	})
}

func (this *CreateEnvironmentHandler) respondInvalidInput(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusBadRequest).JSON(CreateEnvironmentResponseBody{
		Message: EnvironmentInvalidInput,
	})
}

func (this *CreateEnvironmentHandler) respondAlreadyExists(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusOK).JSON(CreateEnvironmentResponseBody{
		Message: EnvironmentAlreadyExists,
	})
}
