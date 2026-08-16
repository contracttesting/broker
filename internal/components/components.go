package components

import (
	"context"
	"fmt"
	"os"

	"github.com/contracttesting/broker/pkg/migrator"
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Components struct {
	Server *fiber.App
	Pool   *pgxpool.Pool
}

func createDatabasePool() *pgxpool.Pool {
	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(fmt.Errorf("Failed to create database pool: %v", err))
	}

	if err := pool.Ping(context.Background()); err != nil {
		panic(fmt.Errorf("Failed to ping database: %v", err))
	}

	return pool
}

// contracts of a few thousand resources do not fit fiber's 4 MiB default
const publishBodyLimit = 8 * 1024 * 1024

func createHttpServer() *fiber.App {
	server := fiber.New(fiber.Config{BodyLimit: publishBodyLimit})

	return server
}

func runMigrations(pool *pgxpool.Pool) {
	migrationsDir := os.Getenv("MIGRATIONS_DIR")

	if migrationsDir == "" {
		migrationsDir = "migrations"
	}

	m := migrator.New(
		pool,
		migrationsDir,
		"public.schema_migrations",
	)

	if err := m.Migrate(); err != nil {
		panic(fmt.Errorf("Failed to run migrations: %v", err))
	}
}

func New() *Components {
	pool := createDatabasePool()
	server := createHttpServer()

	runMigrations(pool)

	return &Components{
		Server: server,
		Pool:   pool,
	}
}
