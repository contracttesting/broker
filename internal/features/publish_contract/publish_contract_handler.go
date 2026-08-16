package publish_contract

import (
	"encoding/json"
	"strings"

	"github.com/contracttesting/broker/internal/dsl"
	"github.com/contracttesting/broker/internal/model"
	"github.com/contracttesting/broker/internal/repository"
	"github.com/gofiber/fiber/v3"
)

type PublishContractHandler struct {
	contractRepository    *repository.ContractRepository
	participantRepository *repository.ParticipantRepository
}

func NewPublishContractHandler(
	contractRepository *repository.ContractRepository,
	participantRepository *repository.ParticipantRepository,
) *PublishContractHandler {
	return &PublishContractHandler{
		contractRepository:    contractRepository,
		participantRepository: participantRepository,
	}
}

func (ctr *PublishContractHandler) Handle(ctx fiber.Ctx) error {
	requestBody := &PublishContractRequestBody{}
	if err := json.Unmarshal(ctx.Body(), requestBody); err != nil {
		return ctr.respondInvalidInput(ctx)
	}

	version := strings.TrimSpace(requestBody.Version)
	if version == "" || len(requestBody.Contract) == 0 {
		return ctr.respondInvalidInput(ctx)
	}

	dslContract := &dsl.Contract{}
	if err := json.Unmarshal(requestBody.Contract, dslContract); err != nil {
		return ctr.respondInvalidInput(ctx)
	}

	participant, exists := ctr.participantRepository.FindByName(ctx.Context(), requestBody.Participant)
	if !exists {
		return ctr.respondParticipantNotFound(ctx)
	}

	contract := model.NewUploadedContract(participant.ID, participant.Name, version, string(requestBody.Contract))
	if err := dslContract.HydrateContract(contract); err != nil {
		return ctr.respondBadRequest(ctx, err)
	}

	if existing, found := ctr.contractRepository.LoadChecksumForVersion(ctx.Context(), contract.ParticipantID, version); found {
		if existing == contract.Checksum() {
			return ctr.respondSuccess(ctx)
		}
		return ctr.respondVersionConflict(ctx)
	}

	// identical content already published under another version: the new version
	// becomes an alias of that snapshot rather than a snapshot of its own
	if ctr.contractRepository.AliasVersionToSnapshot(ctx.Context(), contract.ParticipantID, version, contract.Checksum()) {
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
