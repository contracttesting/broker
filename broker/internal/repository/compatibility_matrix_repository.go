package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/contracttesting/broker/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

const insertCompatibilityMatrixQuery = `
	INSERT INTO compatibility_matrix
		(participant_id, version, counterpart_participant_id, counterpart_version, deployable)
	VALUES
		($1, $2, $3, $4, $5)
	RETURNING id, created_at
`

type CompatibilityMatrixRepository struct {
	pool *pgxpool.Pool
}

func NewCompatibilityMatrixRepository(pool *pgxpool.Pool) *CompatibilityMatrixRepository {
	return &CompatibilityMatrixRepository{pool: pool}
}

func (r *CompatibilityMatrixRepository) Insert(ctx context.Context, row *model.CompatibilityMatrix) {
	var counterpartID sql.NullInt64
	if row.CounterpartParticipantID != 0 {
		counterpartID = sql.NullInt64{Int64: row.CounterpartParticipantID, Valid: true}
	}

	var counterpartVersion sql.NullString
	if row.CounterpartVersion != "" {
		counterpartVersion = sql.NullString{String: row.CounterpartVersion, Valid: true}
	}

	if err := r.pool.QueryRow(
		ctx,
		insertCompatibilityMatrixQuery,
		row.ParticipantID,
		row.Version,
		counterpartID,
		counterpartVersion,
		row.Deployable,
	).Scan(&row.ID, &row.CreatedAt); err != nil {
		panic(fmt.Errorf("error inserting compatibility matrix row: %w", err))
	}
}
