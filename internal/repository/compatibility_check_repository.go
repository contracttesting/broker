package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/contracttesting/broker/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	insertCompatibilityCheckQuery = `
		INSERT INTO compatibility_checks
			(participant_id, version, environment_id, deployable)
		VALUES
			($1, $2, $3, $4)
		RETURNING id, created_at
	`

	insertCompatibilityCheckResultQuery = `
		INSERT INTO compatibility_check_results
			(check_id, counterpart_participant_id, counterpart_version, deployable, reason)
		VALUES
			($1, $2, $3, $4, $5)
		RETURNING id
	`

	loadLatestCompatibilityCheckQuery = `
		SELECT
			id, participant_id, version, environment_id, deployable, created_at
		FROM
			compatibility_checks
		WHERE
			participant_id = $1 AND version = $2 AND environment_id = $3
		ORDER BY
			created_at DESC, id DESC
		LIMIT 1
	`

	loadCompatibilityCheckResultsQuery = `
		SELECT
			r.id, r.check_id, r.counterpart_participant_id, pa.name, r.counterpart_version, r.deployable, r.reason
		FROM
			compatibility_check_results r
		JOIN
			participants pa ON pa.id = r.counterpart_participant_id
		WHERE
			r.check_id = $1
		ORDER BY
			r.id
	`
)

type CompatibilityCheckRepository struct {
	pool *pgxpool.Pool
}

func NewCompatibilityCheckRepository(pool *pgxpool.Pool) *CompatibilityCheckRepository {
	return &CompatibilityCheckRepository{pool: pool}
}

func (r *CompatibilityCheckRepository) Insert(ctx context.Context, check *model.CompatibilityCheck) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		panic(fmt.Errorf("error starting transaction: %w", err))
	}

	defer tx.Rollback(ctx)

	if err := tx.QueryRow(
		ctx,
		insertCompatibilityCheckQuery,
		check.ParticipantID,
		check.Version,
		check.EnvironmentID,
		check.Deployable,
	).Scan(&check.ID, &check.CreatedAt); err != nil {
		panic(fmt.Errorf("error inserting compatibility check: %w", err))
	}

	for i := range check.Results {
		result := &check.Results[i]
		result.CheckID = check.ID

		if err := tx.QueryRow(
			ctx,
			insertCompatibilityCheckResultQuery,
			check.ID,
			result.CounterpartParticipantID,
			result.CounterpartVersion,
			result.Deployable,
			result.Reason,
		).Scan(&result.ID); err != nil {
			panic(fmt.Errorf("error inserting compatibility check result: %w", err))
		}
	}

	if err := tx.Commit(ctx); err != nil {
		panic(fmt.Errorf("error committing transaction: %w", err))
	}
}

func (r *CompatibilityCheckRepository) LoadLatest(
	ctx context.Context,
	participantID int64,
	version string,
	environmentID int64,
) (*model.CompatibilityCheck, bool) {
	check := &model.CompatibilityCheck{}

	err := r.pool.QueryRow(
		ctx,
		loadLatestCompatibilityCheckQuery,
		participantID,
		version,
		environmentID,
	).Scan(
		&check.ID,
		&check.ParticipantID,
		&check.Version,
		&check.EnvironmentID,
		&check.Deployable,
		&check.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false
	}
	if err != nil {
		panic(fmt.Errorf("error loading latest compatibility check: %w", err))
	}

	rows, err := r.pool.Query(ctx, loadCompatibilityCheckResultsQuery, check.ID)
	if err != nil {
		panic(fmt.Errorf("error loading compatibility check results: %w", err))
	}

	defer rows.Close()

	for rows.Next() {
		var result model.CompatibilityCheckResult

		if err := rows.Scan(
			&result.ID,
			&result.CheckID,
			&result.CounterpartParticipantID,
			&result.CounterpartParticipantName,
			&result.CounterpartVersion,
			&result.Deployable,
			&result.Reason,
		); err != nil {
			panic(fmt.Errorf("error scanning compatibility check result: %w", err))
		}

		check.Results = append(check.Results, result)
	}

	return check, true
}
