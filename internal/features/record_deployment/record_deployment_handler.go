package record_deployment

import (
	"context"
	"fmt"

	"github.com/contracttesting/broker/internal/compatibility_checker"
	"github.com/contracttesting/broker/internal/model"
	"github.com/contracttesting/broker/internal/repository"
	"github.com/gofiber/fiber/v3"
	"github.com/guregu/null"
)

type RecordDeploymentHandler struct {
	deploymentRepository         *repository.DeploymentRepository
	participantRepository        *repository.ParticipantRepository
	contractRepository           *repository.ContractRepository
	environmentRepository        *repository.EnvironmentRepository
	compatibilityCheckRepository *repository.CompatibilityCheckRepository
	compatibilityChecker         *compatibility_checker.CompatibilityChecker
}

func NewRecordDeploymentHandler(
	deploymentRepository *repository.DeploymentRepository,
	participantRepository *repository.ParticipantRepository,
	contractRepository *repository.ContractRepository,
	environmentRepository *repository.EnvironmentRepository,
	compatibilityCheckRepository *repository.CompatibilityCheckRepository,
	compatibilityChecker *compatibility_checker.CompatibilityChecker,
) *RecordDeploymentHandler {
	return &RecordDeploymentHandler{
		deploymentRepository:         deploymentRepository,
		participantRepository:        participantRepository,
		contractRepository:           contractRepository,
		environmentRepository:        environmentRepository,
		compatibilityCheckRepository: compatibilityCheckRepository,
		compatibilityChecker:         compatibilityChecker,
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

	if !h.contractRepository.HasContractForVersion(ctx.Context(), participant.ID, requestBody.Version) {
		return h.respondVersionNotFound(ctx)
	}

	environment, exists := h.environmentRepository.FindByName(ctx.Context(), requestBody.Environment)
	if !exists {
		return h.respondEnvironmentNotFound(ctx)
	}

	check, found := h.compatibilityCheckRepository.LoadLatest(ctx.Context(), participant.ID, requestBody.Version, environment.ID)
	if !found {
		return h.respondCheckRequired(ctx, requestBody, map[string]CheckRequiredResult{})
	}

	resolved := h.resolveCounterparts(ctx.Context(), participant.Name, requestBody.Version, environment)

	resultsByCounterpartID := make(map[int64]model.CompatibilityCheckResult, len(check.Results))
	for _, result := range check.Results {
		resultsByCounterpartID[result.CounterpartParticipantID] = result
	}

	drifted := map[string]CheckRequiredResult{}
	failing := map[string]NotDeployableResult{}

	for name, counterpart := range resolved {
		result, covered := resultsByCounterpartID[counterpart.participantID]
		if counterpart.participantID == 0 || !covered {
			drifted[name] = CheckRequiredResult{DeployedVersion: counterpart.version}
			continue
		}

		if result.CounterpartVersion != counterpart.version {
			drifted[name] = CheckRequiredResult{
				CheckedVersion:  result.CounterpartVersion,
				DeployedVersion: counterpart.version,
			}
			continue
		}

		if !result.Deployable {
			failing[name] = NotDeployableResult{
				CounterpartVersion: result.CounterpartVersion,
				Reason:             result.Reason.String,
			}
		}
	}

	if len(drifted) > 0 {
		return h.respondCheckRequired(ctx, requestBody, drifted)
	}

	if len(failing) > 0 && !requestBody.Force {
		return h.respondNotDeployable(ctx, requestBody, failing)
	}

	forced := len(failing) > 0

	h.deploymentRepository.Insert(ctx.Context(), model.NewDeployment(participant, requestBody.Version, environment, forced))

	message := DeploymentRecorded
	if forced {
		message = DeploymentRecordedForced
	}

	return ctx.Status(fiber.StatusOK).JSON(RecordDeploymentResponseBody{
		Message: message,
	})
}

type resolvedCounterpart struct {
	participantID int64
	version       null.String
}

// resolveCounterparts re-runs the compatibility checker's own counterpart
// resolution against the deployments currently in the target environment, so
// drift detection compares the persisted check with exactly the counterpart
// set a fresh check would see.
func (h *RecordDeploymentHandler) resolveCounterparts(
	ctx context.Context,
	participantName string,
	version string,
	environment *model.Environment,
) map[string]resolvedCounterpart {
	resolved := make(map[string]resolvedCounterpart)

	contract, exists := h.contractRepository.GetContractByNameAndVersion(ctx, participantName, version)
	if !exists {
		return resolved
	}

	for name, result := range h.compatibilityChecker.Check(ctx, contract, environment).Results {
		counterpart := resolvedCounterpart{
			participantID: result.IncompatibleCounterpart.ParticipantID,
			version:       result.IncompatibleCounterpart.ParticipantVersion,
		}

		if counterpart.participantID == 0 {
			// the counterpart never published the consumed resource; its
			// participant row may still exist and keep the verdict addressable
			if participant, found := h.participantRepository.FindByName(ctx, name); found {
				counterpart.participantID = participant.ID
			}
		}

		resolved[name] = counterpart
	}

	return resolved
}

func (h *RecordDeploymentHandler) respondCheckRequired(
	ctx fiber.Ctx,
	requestBody *RecordDeploymentRequestBody,
	results map[string]CheckRequiredResult,
) error {
	return ctx.Status(fiber.StatusConflict).JSON(CheckRequiredResponseBody{
		Message: fmt.Sprintf("run can-i-deploy for %s %s against %s first",
			requestBody.Participant, requestBody.Version, requestBody.Environment),
		Reason:  ReasonCompatibilityCheckRequired,
		Results: results,
	})
}

func (h *RecordDeploymentHandler) respondNotDeployable(
	ctx fiber.Ctx,
	requestBody *RecordDeploymentRequestBody,
	results map[string]NotDeployableResult,
) error {
	return ctx.Status(fiber.StatusConflict).JSON(NotDeployableResponseBody{
		Message: fmt.Sprintf("%s %s is not deployable to %s",
			requestBody.Participant, requestBody.Version, requestBody.Environment),
		Reason:  ReasonNotDeployable,
		Results: results,
	})
}

func (h *RecordDeploymentHandler) respondInvalidInput(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusBadRequest).JSON(RecordDeploymentResponseBody{
		Message: DeploymentInvalidInput,
	})
}

func (h *RecordDeploymentHandler) respondEnvironmentNotFound(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusNotFound).JSON(RecordDeploymentResponseBody{
		Message: EnvironmentNotFound,
	})
}

func (h *RecordDeploymentHandler) respondParticipantNotFound(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusNotFound).JSON(RecordDeploymentResponseBody{
		Message: ParticipantNotFound,
	})
}

func (h *RecordDeploymentHandler) respondVersionNotFound(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusNotFound).JSON(RecordDeploymentResponseBody{
		Message: VersionNotFound,
	})
}
