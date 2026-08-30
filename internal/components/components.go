package components

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/contracttesting/broker/pkg/migrator"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Components struct {
	Server *fiber.App
	Pool   *pgxpool.Pool
}

func createDatabasePool() *pgxpool.Pool {
	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(fmt.Errorf("failed to create database pool: %w", err))
	}

	if err := pool.Ping(context.Background()); err != nil {
		panic(fmt.Errorf("failed to ping database: %w", err))
	}

	return pool
}

// contracts of a few thousand resources do not fit fiber's 4 MiB default
const publishBodyLimit = 8 * 1024 * 1024

const internalErrorMessage = "internal error"

type errorResponseBody struct {
	Message string `json:"message"`
}

func respondError(ctx fiber.Ctx, err error) error {
	log.Printf("request %s %s failed: %v", ctx.Method(), ctx.Path(), err)

	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		return ctx.Status(fiberErr.Code).JSON(errorResponseBody{Message: fiberErr.Message})
	}

	return ctx.Status(fiber.StatusInternalServerError).JSON(errorResponseBody{Message: internalErrorMessage})
}

func createHttpServer() *fiber.App {
	server := fiber.New(fiber.Config{
		BodyLimit:    publishBodyLimit,
		ErrorHandler: respondError,
	})

	server.Use(recover.New(recover.Config{EnableStackTrace: true}))

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
		panic(fmt.Errorf("failed to run migrations: %w", err))
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
