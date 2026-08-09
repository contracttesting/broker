package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/contracttesting/broker/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	insertCompatibilityCheckQuery = `
		INSERT INTO compatibility_checks
			(participant_id, contract_id, version, environment_id, deployable)
		VALUES
			($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`

	insertCompatibilityCheckResultQuery = `
		INSERT INTO compatibility_check_results
			(check_id, counterpart_name, counterpart_participant_id, counterpart_version,
			 verdict_contract_id_one, verdict_contract_id_two, deployable)
		VALUES
			($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	insertCompatibilityVerdictQuery = `
		INSERT INTO compatibility_verdicts
			(contract_id_one, contract_id_two, breaks)
		VALUES
			($1, $2, $3)
		ON CONFLICT DO NOTHING
	`

	findCompatibilityVerdictQuery = `
		SELECT
			contract_id_one, contract_id_two, breaks, deployable, created_at
		FROM
			compatibility_verdicts
		WHERE
			contract_id_one = $1 AND contract_id_two = $2
	`
)

// OrderPair is the single canonicalization of a snapshot pair: the lower id first.
func OrderPair(a, b int64) (int64, int64) {
	if a <= b {
		return a, b
	}

	return b, a
}

type CompatibilityRepository struct {
	pool *pgxpool.Pool
}

func NewCompatibilityRepository(pool *pgxpool.Pool) *CompatibilityRepository {
	return &CompatibilityRepository{pool: pool}
}

func (r *CompatibilityRepository) RecordCheck(
	ctx context.Context,
	check *model.CompatibilityCheck,
	results []model.CompatibilityCheckResult,
	newVerdicts []model.CompatibilityVerdict,
) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		panic(fmt.Errorf("error starting transaction: %w", err))
	}

	defer tx.Rollback(ctx)

	r.insertCheck(ctx, tx, check)

	for i := range newVerdicts {
		r.insertVerdict(ctx, tx, &newVerdicts[i])
	}

	for i := range results {
		results[i].CheckID = check.ID
		r.insertCheckResult(ctx, tx, &results[i])
	}

	if err := tx.Commit(ctx); err != nil {
		panic(fmt.Errorf("error committing transaction: %w", err))
	}
}

func (r *CompatibilityRepository) GetVerdict(
	ctx context.Context,
	one, two int64,
) (*model.CompatibilityVerdict, bool) {
	contractIDOne, contractIDTwo := OrderPair(one, two)

	verdict := &model.CompatibilityVerdict{}
	var breaks []byte

	err := r.pool.QueryRow(ctx, findCompatibilityVerdictQuery, contractIDOne, contractIDTwo).Scan(
		&verdict.ContractIDOne,
		&verdict.ContractIDTwo,
		&breaks,
		&verdict.Deployable,
		&verdict.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false
	}
	if err != nil {
		panic(fmt.Errorf("error finding compatibility verdict: %w", err))
	}

	if err := json.Unmarshal(breaks, &verdict.Breaks); err != nil {
		panic(fmt.Errorf("error decoding compatibility verdict breaks: %w", err))
	}

	return verdict, true
}

func (r *CompatibilityRepository) insertCheck(
	ctx context.Context,
	tx pgx.Tx,
	check *model.CompatibilityCheck,
) {
	if err := tx.QueryRow(
		ctx,
		insertCompatibilityCheckQuery,
		check.ParticipantID,
		check.ContractID,
		check.Version,
		check.EnvironmentID,
		check.Deployable,
	).Scan(&check.ID, &check.CreatedAt); err != nil {
		panic(fmt.Errorf("error inserting compatibility check: %w", err))
	}
}

func (r *CompatibilityRepository) insertCheckResult(
	ctx context.Context,
	tx pgx.Tx,
	result *model.CompatibilityCheckResult,
) {
	if err := tx.QueryRow(
		ctx,
		insertCompatibilityCheckResultQuery,
		result.CheckID,
		result.CounterpartName,
		result.CounterpartParticipantID,
		result.CounterpartVersion,
		result.VerdictContractIDOne,
		result.VerdictContractIDTwo,
		result.Deployable,
	).Scan(&result.ID); err != nil {
		panic(fmt.Errorf("error inserting compatibility check result: %w", err))
	}
}

func (r *CompatibilityRepository) insertVerdict(
	ctx context.Context,
	tx pgx.Tx,
	verdict *model.CompatibilityVerdict,
) {
	contractIDOne, contractIDTwo := OrderPair(verdict.ContractIDOne, verdict.ContractIDTwo)

	breaks, err := json.Marshal(verdict.Breaks)
	if err != nil {
		panic(fmt.Errorf("error encoding compatibility verdict breaks: %w", err))
	}

	if _, err := tx.Exec(
		ctx,
		insertCompatibilityVerdictQuery,
		contractIDOne,
		contractIDTwo,
		breaks,
	); err != nil {
		panic(fmt.Errorf("error inserting compatibility verdict: %w", err))
	}
}
