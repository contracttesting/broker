package migrator

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const migrationTimestampLayout = "20060102150405"

var migrationFilenamePattern = regexp.MustCompile(`^(\d{14})_([a-z0-9]+(?:_[a-z0-9]+)*)\.sql$`)

const ensureMigrationsSchemaSQL = `
CREATE SCHEMA IF NOT EXISTS %s;
`

const ensureMigrationsTableSQL = `
CREATE TABLE IF NOT EXISTS %s (
    id SERIAL PRIMARY KEY,
    migration VARCHAR(255) NOT NULL UNIQUE,
    executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
`

const selectMigrationSQL = `SELECT COUNT(*) FROM %s WHERE migration = $1;`

const insertMigrationSQL = `INSERT INTO %s (migration) VALUES ($1);`

type Migrator struct {
	pool            *pgxpool.Pool
	migrationsDir   string
	migrationsTable string
}

func New(
	pool *pgxpool.Pool,
	migrationsDir, migrationsTable string,
) *Migrator {
	return &Migrator{
		pool:            pool,
		migrationsDir:   migrationsDir,
		migrationsTable: migrationsTable,
	}
}

func (m *Migrator) Migrate() error {
	if err := m.ensureMigrationsTable(); err != nil {
		return fmt.Errorf("error ensuring migrations table: %w", err)
	}

	migrationFiles, err := m.getMigrationFiles()
	if err != nil {
		return fmt.Errorf("error reading migrations directory: %w", err)
	}

	ctx := context.Background()
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error starting migrations transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, migrationFile := range migrationFiles {
		applied, err := m.isMigrationApplied(ctx, tx, migrationFile)
		if err != nil {
			return fmt.Errorf(
				"error checking if migration %s is applied: %w",
				migrationFile,
				err,
			)
		}

		if applied {
			continue
		}

		if err := m.applyMigration(ctx, tx, migrationFile); err != nil {
			return err
		}

		log.Printf("migrated %s", migrationFile)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error committing migrations: %w", err)
	}

	return nil
}

func (m *Migrator) ensureMigrationsTable() error {
	var sql string

	// only a qualified name identifies a schema to create; an unqualified one resolves
	// through search_path and must not be mistaken for a schema of its own
	if schema, _, qualified := strings.Cut(m.migrationsTable, "."); qualified {
		sql = fmt.Sprintf(ensureMigrationsSchemaSQL, schema)
	}

	sql += fmt.Sprintf(ensureMigrationsTableSQL, m.migrationsTable)

	if _, err := m.pool.Exec(context.Background(), sql); err != nil {
		return err
	}

	return nil
}

func (m *Migrator) getMigrationFiles() ([]string, error) {
	entries, err := os.ReadDir(m.migrationsDir)
	if err != nil {
		return nil, err
	}

	var migrationFiles []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		assertValidMigrationFilename(entry.Name())
		migrationFiles = append(migrationFiles, filepath.Join(m.migrationsDir, entry.Name()))
	}

	return migrationFiles, nil
}

func assertValidMigrationFilename(name string) {
	matches := migrationFilenamePattern.FindStringSubmatch(name)
	if matches == nil {
		panic(fmt.Sprintf(
			"migrator: invalid migration filename %q: expected format YYYYMMDDHHMMSS_subject.sql (e.g. 20260520143022_add_users_table.sql)",
			name,
		))
	}
	if _, err := time.Parse(migrationTimestampLayout, matches[1]); err != nil {
		panic(fmt.Sprintf(
			"migrator: invalid migration timestamp in %q: %v",
			name, err,
		))
	}
}

func (m *Migrator) isMigrationApplied(ctx context.Context, tx pgx.Tx, migrationFile string) (bool, error) {
	sql := fmt.Sprintf(selectMigrationSQL, m.migrationsTable)

	var count int
	if err := tx.QueryRow(ctx, sql, migrationFile).Scan(&count); err != nil {
		return false, err
	}

	return count > 0, nil
}

func (m *Migrator) applyMigration(ctx context.Context, tx pgx.Tx, migrationFile string) error {
	contents, err := os.ReadFile(migrationFile)
	if err != nil {
		return fmt.Errorf("error reading migration file %s: %w", migrationFile, err)
	}

	if _, err := tx.Exec(ctx, string(contents)); err != nil {
		return fmt.Errorf("error executing migration %s: %w", migrationFile, err)
	}

	sql := fmt.Sprintf(insertMigrationSQL, m.migrationsTable)
	if _, err := tx.Exec(ctx, sql, migrationFile); err != nil {
		return fmt.Errorf("error recording migration %s: %w", migrationFile, err)
	}

	return nil
}
