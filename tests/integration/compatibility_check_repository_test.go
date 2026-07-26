package integration_test

import (
	"context"

	"github.com/contracttesting/broker/internal/model"
	"github.com/contracttesting/broker/internal/repository"
	"github.com/guregu/null"
)

func (s *IntegrationSuite) TestCompatibilityCheckRepository_LoadLatestReturnsMostRecentCheckWithResults() {
	ctx := context.Background()

	participantRepository := repository.NewParticipantRepository(s.Pool)
	app := &model.Participant{Name: "app"}
	participantRepository.Create(ctx, app)
	api := &model.Participant{Name: "api"}
	participantRepository.Create(ctx, api)
	billing := &model.Participant{Name: "billing"}
	participantRepository.Create(ctx, billing)

	environmentRepository := repository.NewEnvironmentRepository(s.Pool)
	production := &model.Environment{Name: "production"}
	environmentRepository.Create(ctx, production)

	checkRepository := repository.NewCompatibilityCheckRepository(s.Pool)

	first := &model.CompatibilityCheck{
		ParticipantID: app.ID,
		Version:       "v1",
		EnvironmentID: production.ID,
		Deployable:    false,
		Results: []model.CompatibilityCheckResult{
			{
				CounterpartParticipantID: api.ID,
				CounterpartVersion:       null.StringFrom("v1"),
				Deployable:               false,
				Reason:                   null.StringFrom("property_type_mismatch"),
			},
		},
	}
	checkRepository.Insert(ctx, first)

	second := &model.CompatibilityCheck{
		ParticipantID: app.ID,
		Version:       "v1",
		EnvironmentID: production.ID,
		Deployable:    false,
		Results: []model.CompatibilityCheckResult{
			{
				CounterpartParticipantID: api.ID,
				CounterpartVersion:       null.StringFrom("v2"),
				Deployable:               true,
			},
			{
				CounterpartParticipantID: billing.ID,
				Deployable:               false,
				Reason:                   null.StringFrom("provider_resource_not_deployed_in_environment"),
			},
		},
	}
	checkRepository.Insert(ctx, second)

	got, found := checkRepository.LoadLatest(ctx, app.ID, "v1", production.ID)
	s.Require().True(found)
	s.Equal(second.ID, got.ID)
	s.False(got.Deployable)

	s.Require().Len(got.Results, 2)

	s.Equal(api.ID, got.Results[0].CounterpartParticipantID)
	s.Equal("api", got.Results[0].CounterpartParticipantName)
	s.Equal(null.StringFrom("v2"), got.Results[0].CounterpartVersion)
	s.True(got.Results[0].Deployable)
	s.False(got.Results[0].Reason.Valid)

	s.Equal(billing.ID, got.Results[1].CounterpartParticipantID)
	s.Equal("billing", got.Results[1].CounterpartParticipantName)
	s.False(got.Results[1].CounterpartVersion.Valid)
	s.False(got.Results[1].Deployable)
	s.Equal(null.StringFrom("provider_resource_not_deployed_in_environment"), got.Results[1].Reason)
}

func (s *IntegrationSuite) TestCompatibilityCheckRepository_LoadLatestNotFound() {
	ctx := context.Background()

	participantRepository := repository.NewParticipantRepository(s.Pool)
	app := &model.Participant{Name: "app"}
	participantRepository.Create(ctx, app)

	environmentRepository := repository.NewEnvironmentRepository(s.Pool)
	production := &model.Environment{Name: "production"}
	environmentRepository.Create(ctx, production)

	got, found := repository.NewCompatibilityCheckRepository(s.Pool).LoadLatest(ctx, app.ID, "v1", production.ID)
	s.False(found)
	s.Nil(got)
}
