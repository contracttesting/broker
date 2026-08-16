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
	if version == "" || len(requestBody.Contracts) == 0 {
		return ctr.respondInvalidInput(ctx)
	}

	fragments := make([]dsl.Fragment, 0, len(requestBody.Contracts))
	for _, uploaded := range requestBody.Contracts {
		if strings.TrimSpace(uploaded.Source) == "" {
			return ctr.respondInvalidInput(ctx)
		}

		parsed, err := parseFragment(uploaded)
		if err != nil {
			return ctr.respondBadRequest(ctx, err)
		}

		fragments = append(fragments, dsl.Fragment{Source: uploaded.Source, Contract: parsed})
	}

	participant, exists := ctr.participantRepository.FindByName(ctx.Context(), requestBody.Participant)
	if !exists {
		return ctr.respondParticipantNotFound(ctx)
	}

	// the uploaded files are stamped on the version verbatim, so a version reconstructs
	// letter by letter what was published under it
	contractContent, _ := json.Marshal(requestBody.Contracts)

	contract := model.NewUploadedContract(participant.ID, participant.Name, version, string(contractContent))
	if err := dsl.HydrateFragments(fragments, contract); err != nil {
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
