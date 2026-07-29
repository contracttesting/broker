package integration_test

import (
	"context"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/contracttesting/broker/internal"
	"github.com/contracttesting/broker/internal/components"
	"github.com/contracttesting/broker/pkg/rootpath"
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationSuite))
}

type IntegrationSuite struct {
	suite.Suite

	container  *postgres.PostgresContainer
	Components *components.Components
	Pool       *pgxpool.Pool
}

func (s *IntegrationSuite) SetupSuite() {
	ctx := context.Background()

	container, err := postgres.Run(ctx,
		"postgres:16.6-alpine",
		postgres.WithDatabase("contracttests"),
		postgres.WithUsername("contracttests"),
		postgres.WithPassword("s3cr3t"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	s.Require().NoError(err)
	s.container = container

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	s.Require().NoError(err)

	s.Require().NoError(os.Setenv("DATABASE_URL", connStr))
	s.Require().NoError(os.Setenv("MIGRATIONS_DIR", filepath.Join(rootpath.Discover(), "migrations")))

	s.Components = internal.Run()
	s.Pool = s.Components.Pool
}

func (s *IntegrationSuite) TearDownSuite() {
	if s.Pool != nil {
		s.Pool.Close()
	}
	if s.container != nil {
		_ = s.container.Terminate(context.Background())
	}
}

func (s *IntegrationSuite) SetupTest() {
	_, err := s.Pool.Exec(context.Background(),
		`TRUNCATE 
			compatibility_matrix,
			deployments,
			property_versions,
			resource_versions,
			properties,
			resources,
			contract_versions,
			contracts,
			participants,
			environments
			RESTART IDENTITY CASCADE`,
	)

	s.Require().NoError(err)
}

func (s *IntegrationSuite) post(path, body string) (status int, response string) {
	req := httptest.NewRequest("POST", path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.Components.Server.Test(req, fiber.TestConfig{Timeout: 10 * time.Second})
	s.Require().NoError(err)
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)

	return resp.StatusCode, string(bytes)
}

func (s *IntegrationSuite) countRows(table string) int {
	var count int
	err := s.Pool.QueryRow(context.Background(),
		fmt.Sprintf("SELECT count(*) FROM %s", table),
	).Scan(&count)
	s.Require().NoError(err)
	return count
}
