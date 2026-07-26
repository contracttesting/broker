package integration_test

import (
	"context"
	"net/http"

	"github.com/contracttesting/broker/internal/repository"
)

const stagingEnvBody = `{"participant":"staging"}`

func (s *IntegrationSuite) insertDeploymentAt(participantID int64, version string, environmentID int64, deployedAt string) {
	_, err := s.Pool.Exec(context.Background(),
		`WITH prior AS (
		     SELECT version FROM deployments
		     WHERE participant_id = $1 AND environment_id = $3
		 )
		 INSERT INTO deployments (participant_id, version, environment_id, rollback, deployed_at)
		 SELECT $1, $2, $3,
		        EXISTS (SELECT 1 FROM prior WHERE version = $2),
		        $4::timestamptz`,
		participantID, version, environmentID, deployedAt,
	)
	s.Require().NoError(err)
}

func (s *IntegrationSuite) TestCurrentVersionInEnv_RollbackPicksLatestRowEvenIfOlderVersion() {
	s.seedApiParticipantContractAndProductionEnv()

	status, _ := s.post("/api/contracts", `{"participant":"api","version":"v2","contract":`+apiV1ContractBody+`}`)
	s.Require().Equal(http.StatusOK, status)

	participantID := s.lookupParticipantID("api")
	productionID := s.lookupEnvironmentID("production")

	s.insertDeploymentAt(participantID, "v1", productionID, "2026-05-01T00:00:00Z")
	s.insertDeploymentAt(participantID, "v2", productionID, "2026-05-10T00:00:00Z")
	s.insertDeploymentAt(participantID, "v1", productionID, "2026-05-15T00:00:00Z")

	repo := repository.NewDeploymentRepository(s.Pool)
	version, ok := repo.CurrentVersionInEnv(context.Background(), participantID, productionID)
	s.True(ok)
	s.Equal("v1", version)
}

func (s *IntegrationSuite) TestCurrentVersionInEnv_NoRowsReturnsNotFound() {
	s.seedApiParticipantContractAndProductionEnv()

	participantID := s.lookupParticipantID("api")
	productionID := s.lookupEnvironmentID("production")

	repo := repository.NewDeploymentRepository(s.Pool)
	version, ok := repo.CurrentVersionInEnv(context.Background(), participantID, productionID)
	s.False(ok)
	s.Equal("", version)
}

func (s *IntegrationSuite) TestCurrentVersionInEnv_ScopedPerEnvironment() {
	s.seedApiParticipantContractAndProductionEnv()

	status, _ := s.post("/api/environments", stagingEnvBody)
	s.Require().Equal(http.StatusOK, status)

	status, _ = s.post("/api/contracts", `{"participant":"api","version":"v2","contract":`+apiV1ContractBody+`}`)
	s.Require().Equal(http.StatusOK, status)

	participantID := s.lookupParticipantID("api")
	productionID := s.lookupEnvironmentID("production")
	stagingID := s.lookupEnvironmentID("staging")

	s.insertDeploymentAt(participantID, "v1", productionID, "2026-05-01T00:00:00Z")
	s.insertDeploymentAt(participantID, "v2", stagingID, "2026-05-02T00:00:00Z")

	repo := repository.NewDeploymentRepository(s.Pool)

	prodVersion, prodOk := repo.CurrentVersionInEnv(context.Background(), participantID, productionID)
	s.True(prodOk)
	s.Equal("v1", prodVersion)

	stagingVersion, stagingOk := repo.CurrentVersionInEnv(context.Background(), participantID, stagingID)
	s.True(stagingOk)
	s.Equal("v2", stagingVersion)
}
