package publish_contract

import (
	"encoding/json"
	"strings"

	"github.com/contracttesting/broker/internal/builder"
	"github.com/contracttesting/broker/internal/dsl"
	"github.com/contracttesting/broker/internal/features/publish_contract/validator"
	"github.com/contracttesting/broker/internal/model"
	"github.com/contracttesting/broker/internal/repository"
	"github.com/gofiber/fiber/v3"
)

type PublishContractHandler struct {
	contractRepository    *repository.ContractRepository
	participantRepository *repository.ParticipantRepository
	validator             *validator.DslValidator
}

func NewPublishContractHandler(
	contractRepository *repository.ContractRepository,
	participantRepository *repository.ParticipantRepository,
	contractValidator *validator.DslValidator,
) *PublishContractHandler {
	return &PublishContractHandler{
		contractRepository:    contractRepository,
		participantRepository: participantRepository,
		validator:             contractValidator,
	}
}

func (ctr *PublishContractHandler) Handle(ctx fiber.Ctx) error {
	requestBody := &PublishContractRequestBody{}
	if err := json.Unmarshal(ctx.Body(), requestBody); err != nil {
		return ctr.respondInvalidInput(ctx)
	}

	serviceName := strings.TrimSpace(requestBody.ServiceName)
	version := strings.TrimSpace(requestBody.Version)
	if serviceName == "" || version == "" || len(requestBody.Contracts) == 0 {
		return ctr.respondInvalidInput(ctx)
	}

	contractFragments := make([]dsl.Fragment, 0, len(requestBody.Contracts))
	for _, uploaded := range requestBody.Contracts {
		if strings.TrimSpace(uploaded.Source) == "" {
			return ctr.respondInvalidInput(ctx)
		}

		contractDsl, err := parseFragmentContentToContractDsl(uploaded)
		if err != nil {
			return ctr.respondBadRequest(ctx, err)
		}

		contractFragments = append(contractFragments, dsl.Fragment{Source: uploaded.Source, Contract: contractDsl})
	}

	participant, exists := ctr.participantRepository.FindByName(ctx.Context(), serviceName)
	if !exists {
		return ctr.respondParticipantNotFound(ctx)
	}

	if violations := ctr.validator.Validate(contractFragments); len(violations) > 0 {
		return ctr.respondValidationFailed(ctx, violations)
	}

	dsl.Normalize(contractFragments)

	contractContent, _ := json.Marshal(requestBody.Contracts)

	contract := model.NewUploadedContract(participant.ID, participant.Name, version, string(contractContent))
	if err := builder.Hydrate(contractFragments, contract); err != nil {
		return ctr.respondPublishFailed(ctx)
	}

	if existing, found := ctr.contractRepository.LoadChecksumForVersion(ctx.Context(), contract.ParticipantID, version); found {
		if existing == contract.Checksum() {
			return ctr.respondSuccess(ctx)
		}
		return ctr.respondVersionConflict(ctx)
	}

	if ctr.contractRepository.AliasVersionToSnapshot(ctx.Context(), contract) {
		return ctr.respondSuccess(ctx)
	}

	ctr.upsert(ctx, contract)

	return ctr.respondSuccess(ctx)
}

func (ctr *PublishContractHandler) respondParticipantNotFound(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusNotFound).JSON(PublishContractResponseBody{
		Message: ContractParticipantNotFound,
	})
}

func (ctr *PublishContractHandler) upsert(ctx fiber.Ctx, contract *model.UploadedContract) {
	if ctr.contractRepository.HasContractsForParticipant(ctx.Context(), contract.ParticipantID) {
		ctr.contractRepository.Update(ctx.Context(), contract)

		return
	}

	ctr.contractRepository.Create(ctx.Context(), contract)
}

func (ctr *PublishContractHandler) respondBadRequest(ctx fiber.Ctx, err error) error {
	return ctx.Status(fiber.StatusBadRequest).JSON(PublishContractResponseBody{
		Message: err.Error(),
	})
}

func (ctr *PublishContractHandler) respondValidationFailed(ctx fiber.Ctx, violations []string) error {
	return ctx.Status(fiber.StatusBadRequest).JSON(PublishContractValidationResponseBody{
		Message:    ContractValidationFailed,
		Violations: violations,
	})
}

func (ctr *PublishContractHandler) respondPublishFailed(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusInternalServerError).JSON(PublishContractResponseBody{
		Message: ContractPublishFailed,
	})
}

func (ctr *PublishContractHandler) respondInvalidInput(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusBadRequest).JSON(PublishContractResponseBody{
		Message: ContractInvalidInput,
	})
}

func (ctr *PublishContractHandler) respondVersionConflict(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusConflict).JSON(PublishContractResponseBody{
		Message: ContractVersionConflict,
	})
}

func (ctr *PublishContractHandler) respondSuccess(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusOK).JSON(PublishContractResponseBody{
		Message: ContractPublishSuccessful,
	})
}
