package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/contracttesting/broker/internal/contract_differ"
	"github.com/contracttesting/broker/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrProviderResourceNotFound = errors.New("provider resource not found")
)

const (
	hasContractsForParticipantQuery = `
		SELECT EXISTS(SELECT 1 FROM contracts WHERE participant_id = $1)
	`

	hasContractForVersionQuery = `
		SELECT EXISTS(SELECT 1 FROM contracts WHERE participant_id = $1 AND version = $2)
	`

	loadChecksumForVersionQuery = `
		SELECT checksum FROM contracts WHERE participant_id = $1 AND version = $2
	`

	insertContractQuery = `
		INSERT INTO contracts
			(participant_id, version, checksum, raw_payload)
		VALUES
			($1, $2, $3, $4)
		RETURNING id
	`

	insertResourceQuery = `
		INSERT INTO resources
			(
				participant_id,
				direction,
				kind,
				consumed_provider,
				endpoint,
				method,
				response_status_code,
				provider_hash,
				consumer_hash
			)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	findResourceIDByProviderHashQuery = `
		SELECT id FROM resources WHERE direction = 'provides' AND provider_hash = $1
	`

	findResourceIDByConsumerHashQuery = `
		SELECT id FROM resources WHERE direction = 'consumes' AND consumer_hash = $1
	`

	insertPropertyQuery = `
		INSERT INTO properties
			(resource_id, path)
		VALUES
			($1, $2)
		RETURNING id
	`

	findPropertyIDByResourceAndPathQuery = `
		SELECT id FROM properties WHERE resource_id = $1 AND path = $2
	`

	insertPropertyVersionQuery = `
		INSERT INTO property_versions
			(property_id, contract_id, type, optional, change_type)
		VALUES
			($1, $2, $3, $4, $5)
	`

	insertResourceVersionQuery = `
		INSERT INTO resource_versions
			(resource_id, contract_id, change_type)
		VALUES
			($1, $2, $3)
	`

	findContractTreeQuery = `
		SELECT
			c.id,
			c.version,
			c.raw_payload,
			c.created_at,
			pa.id,
			pa.name,
			r.id,
			r.direction,
			r.kind,
			r.consumed_provider,
			r.endpoint,
			r.method,
			r.response_status_code,
			r.provider_hash,
			r.consumer_hash,
			r.created_at,
			p.id,
			p.path,
			pv.type,
			pv.optional,
			pv.change_type
		FROM contracts c
		JOIN participants pa ON pa.id = c.participant_id
		JOIN resources r ON r.participant_id = c.participant_id
		JOIN LATERAL (
			SELECT change_type
			FROM resource_versions
			WHERE resource_id = r.id AND contract_id <= c.id
			ORDER BY contract_id DESC
			LIMIT 1
		) rv ON true
		JOIN properties p ON p.resource_id = r.id
		JOIN LATERAL (
			SELECT type, optional, change_type
			FROM property_versions
			WHERE property_id = p.id AND contract_id <= c.id
			ORDER BY contract_id DESC
			LIMIT 1
		) pv ON true
		WHERE pa.name = $1
		  AND c.id = (SELECT MAX(id) FROM contracts WHERE participant_id = pa.id)
		  AND rv.change_type = 'added'
		ORDER BY r.id
	`

	findContractTreeByNameAndVersionQuery = `
		SELECT
			c.id,
			c.version,
			c.raw_payload,
			c.created_at,
			pa.id,
			pa.name,
			r.id,
			r.direction,
			r.kind,
			r.consumed_provider,
			r.endpoint,
			r.method,
			r.response_status_code,
			r.provider_hash,
			r.consumer_hash,
			r.created_at,
			p.id,
			p.path,
			pv.type,
			pv.optional,
			pv.change_type
		FROM contracts c
		JOIN participants pa ON pa.id = c.participant_id
		JOIN resources r ON r.participant_id = c.participant_id
		JOIN LATERAL (
			SELECT change_type
			FROM resource_versions
			WHERE resource_id = r.id AND contract_id <= c.id
			ORDER BY contract_id DESC
			LIMIT 1
		) rv ON true
		JOIN properties p ON p.resource_id = r.id
		JOIN LATERAL (
			SELECT type, optional, change_type
			FROM property_versions
			WHERE property_id = p.id AND contract_id <= c.id
			ORDER BY contract_id DESC
			LIMIT 1
		) pv ON true
		WHERE pa.name = $1
		  AND c.version = $2
		  AND rv.change_type = 'added'
		ORDER BY r.id
	`

	findCurrentConsumersOfProviderInEnvQuery = `
		WITH current_deployments AS (
			SELECT DISTINCT ON (d.participant_id)
				d.participant_id, d.version, c.id AS contract_id
			FROM deployments d
			JOIN contracts c ON c.participant_id = d.participant_id AND c.version = d.version
			WHERE d.environment_id = $2
			ORDER BY d.participant_id, d.deployed_at DESC
		)
		SELECT DISTINCT cd.participant_id, pa.name, cd.version
		FROM current_deployments cd
		JOIN resources r ON r.participant_id = cd.participant_id
		JOIN participants pa ON pa.id = cd.participant_id
		JOIN LATERAL (
			SELECT change_type
			FROM resource_versions
			WHERE resource_id = r.id AND contract_id <= cd.contract_id
			ORDER BY contract_id DESC
			LIMIT 1
		) rv ON true
		WHERE r.direction = 'consumes'
		  AND r.provider_hash = $1
		  AND rv.change_type = 'added'
	`

	findResourcesByDirectionAndProviderHashQuery = `
		SELECT
			r.id,
			pa.id,
			pa.name,
			r.direction,
			r.kind,
			r.consumed_provider,
			r.endpoint,
			r.method,
			r.response_status_code,
			r.provider_hash,
			r.consumer_hash,
			r.created_at,
			p.path,
			pv.type,
			pv.optional,
			pv.change_type,
			dep.version
		FROM
			resources r
		JOIN
			participants pa ON pa.id = r.participant_id
		JOIN LATERAL (
			SELECT version
			FROM deployments
			WHERE participant_id = r.participant_id
			  AND environment_id = $3
			ORDER BY deployed_at DESC
			LIMIT 1
		) dep ON true
		JOIN LATERAL (
			SELECT change_type
			FROM resource_versions
			WHERE resource_id = r.id
			  AND contract_id <= (SELECT MAX(id) FROM contracts WHERE participant_id = r.participant_id)
			ORDER BY contract_id DESC
			LIMIT 1
		) rv ON true
		LEFT JOIN
			properties p ON p.resource_id = r.id
		LEFT JOIN LATERAL (
			SELECT
				type, optional, change_type
			FROM
				property_versions
			WHERE
				property_id = p.id
			ORDER BY contract_id DESC
			LIMIT 1
		) pv ON true
		WHERE
			r.direction = $1
		AND
			r.provider_hash = $2
		AND
			rv.change_type = 'added'
	`

	loadProviderResourceWithDeploymentsQuery = `
		SELECT
			r.id,
			pa.id,
			pa.name,
			r.direction,
			r.kind,
			r.consumed_provider,
			r.endpoint,
			r.method,
			r.response_status_code,
			r.provider_hash,
			r.consumer_hash,
			r.created_at,
			p.path,
			pv.type,
			pv.optional,
			pv.change_type,
			e.name,
			dep.version
		FROM
			resources r
		JOIN
			participants pa ON pa.id = r.participant_id
		JOIN LATERAL (
			SELECT change_type
			FROM resource_versions
			WHERE resource_id = r.id
			  AND contract_id <= (SELECT MAX(id) FROM contracts WHERE participant_id = r.participant_id)
			ORDER BY contract_id DESC
			LIMIT 1
		) rv ON true
		LEFT JOIN
			properties p ON p.resource_id = r.id
		LEFT JOIN LATERAL (
			SELECT
				type, optional, change_type
			FROM
				property_versions
			WHERE
				property_id = p.id
			ORDER BY contract_id DESC
			LIMIT 1
		) pv ON true
		LEFT JOIN LATERAL (
			SELECT DISTINCT ON (environment_id) environment_id, version
			FROM deployments
			WHERE participant_id = r.participant_id
			ORDER BY environment_id, deployed_at DESC
		) dep ON true
		LEFT JOIN
			environments e ON e.id = dep.environment_id
		WHERE
			r.direction = $1
		AND
			r.provider_hash = $2
		AND
			rv.change_type = 'added'
	`
)

type ContractRepository struct {
	pool *pgxpool.Pool
}

func NewContractRepository(pool *pgxpool.Pool) *ContractRepository {
	return &ContractRepository{pool: pool}
}

func (r *ContractRepository) HasContractsForParticipant(ctx context.Context, participantID int64) bool {
	var exists bool

	if err := r.pool.QueryRow(ctx, hasContractsForParticipantQuery, participantID).Scan(&exists); err != nil {
		panic(fmt.Errorf("error checking contracts for participant: %w", err))
	}

	return exists
}

func (r *ContractRepository) HasContractForVersion(ctx context.Context, participantID int64, version string) bool {
	var exists bool

	if err := r.pool.QueryRow(ctx, hasContractForVersionQuery, participantID, version).Scan(&exists); err != nil {
		panic(fmt.Errorf("error checking contract version: %w", err))
	}

	return exists
}

func (r *ContractRepository) LoadChecksumForVersion(ctx context.Context, participantID int64, version string) (string, bool) {
	var checksum string

	err := r.pool.QueryRow(ctx, loadChecksumForVersionQuery, participantID, version).Scan(&checksum)
	if err == nil {
		return checksum, true
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false
	}
	panic(fmt.Errorf("error loading checksum for version: %w", err))
}

func (r *ContractRepository) Create(
	ctx context.Context,
	contract *model.Contract,
) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		panic(fmt.Errorf("error starting transaction: %w", err))
	}

	defer tx.Rollback(ctx)

	r.insertContract(ctx, tx, contract)

	for _, resource := range contract.Resources {
		r.insertResource(ctx, tx, &resource)
		r.insertResourceVersion(ctx, tx, newInsertResourceVersionRowAdded(contract, resource))

		for _, property := range resource.Properties {
			r.insertNewProperty(ctx, tx, resource.ID, &property)
			r.insertPropertyVersion(ctx, tx, newInsertPropertyVersionRowAdded(contract, property))
		}
	}

	if err := tx.Commit(ctx); err != nil {
		panic(fmt.Errorf("error committing transaction: %w", err))
	}
}

func (r *ContractRepository) Update(
	ctx context.Context,
	next *model.Contract,
) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		panic(fmt.Errorf("error starting transaction: %w", err))
	}

	defer tx.Rollback(ctx)

	contract := r.LoadLatestContractByName(ctx, next.Participant.Name)

	diff := contract_differ.Diff(contract, next)

	if len(diff.Resources) == 0 {
		if err := tx.Commit(ctx); err != nil {
			panic(fmt.Errorf("error committing transaction: %w", err))
		}

		return
	}

	next.ID = contract.ID

	r.insertContract(ctx, tx, next)

	for _, resourceChange := range diff.Resources {
		switch resourceChange.Kind {
		case contract_differ.ChangeAdded:
			resource := resourceChange.Resource
			r.insertResource(ctx, tx, &resource)
			r.insertResourceVersion(ctx, tx, newInsertResourceVersionRowAdded(next, resource))

			for _, property := range resource.Properties {
				r.insertNewProperty(ctx, tx, resource.ID, &property)
				r.insertPropertyVersion(ctx, tx, newInsertPropertyVersionRowAdded(next, property))
			}

		case contract_differ.ChangeModified:
			resource := contract.Resources[resourceChange.Resource.PrimaryHash()]

			for _, propertyChange := range resourceChange.Properties {
				switch propertyChange.Kind {
				case contract_differ.ChangeAdded:
					property := propertyChange.After
					r.insertNewProperty(ctx, tx, resource.ID, &property)
					r.insertPropertyVersion(ctx, tx, newInsertPropertyVersionRowAdded(next, property))

				case contract_differ.ChangeModified:
					property := propertyChange.After
					property.ID = resource.Properties[propertyChange.After.Path].ID
					r.insertPropertyVersion(ctx, tx, newInsertPropertyVersionRowModified(next, property))

				case contract_differ.ChangeRemoved:
					property := resource.Properties[propertyChange.Before.Path]
					r.insertPropertyVersion(ctx, tx, newInsertPropertyVersionRowRemoved(next, property))
				}
			}

		case contract_differ.ChangeRemoved:
			resource := contract.Resources[resourceChange.Resource.PrimaryHash()]
			r.insertResourceVersion(ctx, tx, newInsertResourceVersionRowRemoved(next, resource))

			for _, property := range resource.Properties {
				r.insertPropertyVersion(ctx, tx, newInsertPropertyVersionRowRemoved(next, property))
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		panic(fmt.Errorf("error committing transaction: %w", err))
	}
}

func (r *ContractRepository) insertContract(
	ctx context.Context,
	tx pgx.Tx,
	contract *model.Contract,
) {
	if err := tx.QueryRow(
		ctx,
		insertContractQuery,
		contract.ParticipantID(),
		contract.Version,
		contract.Checksum(),
		contract.RawContract,
	).Scan(&contract.ID); err != nil {
		panic(fmt.Errorf("error inserting contract: %w", err))
	}
}

func (r *ContractRepository) insertResource(
	ctx context.Context,
	tx pgx.Tx,
	resource *model.Resource,
) {
	if id, ok := r.findExistingResourceID(ctx, tx, resource); ok {
		resource.ID = id
		return
	}

	statusCode := sql.NullString{
		String: resource.ResponseStatusCode,
		Valid:  resource.ResponseStatusCode != "",
	}

	provider := sql.NullString{
		String: resource.ConsumedProvider,
		Valid:  resource.ConsumedProvider != "",
	}

	providerHash := sql.NullString{
		String: resource.ProviderHash(),
		Valid:  resource.ProviderHash() != "",
	}

	consumerHash := sql.NullString{
		String: resource.ConsumerHash(),
		Valid:  resource.ConsumerHash() != "",
	}

	if err := tx.QueryRow(
		ctx,
		insertResourceQuery,
		resource.ParticipantID(),
		resource.Direction.String(),
		resource.Kind.String(),
		provider,
		resource.Endpoint,
		resource.Method,
		statusCode,
		providerHash,
		consumerHash,
	).Scan(&resource.ID); err != nil {
		panic(fmt.Errorf("error inserting resource: %w", err))
	}
}

func (r *ContractRepository) findExistingResourceID(
	ctx context.Context,
	tx pgx.Tx,
	resource *model.Resource,
) (int64, bool) {
	var query, hash string
	if resource.Direction == model.Provides {
		query = findResourceIDByProviderHashQuery
		hash = resource.ProviderHash()
	} else {
		query = findResourceIDByConsumerHashQuery
		hash = resource.ConsumerHash()
	}

	var id int64
	err := tx.QueryRow(ctx, query, hash).Scan(&id)
	if err == nil {
		return id, true
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false
	}
	panic(fmt.Errorf("error looking up existing resource: %w", err))
}

func (r *ContractRepository) insertNewProperty(
	ctx context.Context,
	tx pgx.Tx,
	resourceID int64,
	property *model.Property,
) {
	var existingID int64
	err := tx.QueryRow(ctx, findPropertyIDByResourceAndPathQuery, resourceID, property.Path).Scan(&existingID)
	if err == nil {
		property.ID = existingID
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		panic(fmt.Errorf("error looking up existing property: %w", err))
	}

	if err := tx.QueryRow(
		ctx,
		insertPropertyQuery,
		resourceID,
		property.Path,
	).Scan(&property.ID); err != nil {
		panic(fmt.Errorf("error inserting property: %w", err))
	}
}

func (r *ContractRepository) insertPropertyVersion(
	ctx context.Context,
	tx pgx.Tx,
	row *insertPropertyVersionRow,
) {
	if _, err := tx.Exec(
		ctx,
		insertPropertyVersionQuery,
		row.PropertyID,
		row.ContractID,
		row.Type,
		row.Optional,
		row.ChangeType,
	); err != nil {
		panic(fmt.Errorf("error inserting property version: %w", err))
	}
}

func (r *ContractRepository) insertResourceVersion(
	ctx context.Context,
	tx pgx.Tx,
	row *insertResourceVersionRow,
) {
	if _, err := tx.Exec(
		ctx,
		insertResourceVersionQuery,
		row.ResourceID,
		row.ContractID,
		row.ChangeType,
	); err != nil {
		panic(fmt.Errorf("error inserting resource version: %w", err))
	}
}

func (r *ContractRepository) LoadLatestContractByName(
	ctx context.Context,
	participantName string,
) *model.Contract {
	rows, err := r.pool.Query(ctx, findContractTreeQuery, participantName)

	if err != nil {
		panic(fmt.Errorf("error loading contract tree: %w", err))
	}

	defer rows.Close()

	contract, _ := scanContractTree(rows)
	return contract
}

func (r *ContractRepository) LoadContractByNameAndVersion(
	ctx context.Context,
	participantName string,
	version string,
) (*model.Contract, bool) {
	rows, err := r.pool.Query(ctx, findContractTreeByNameAndVersionQuery, participantName, version)
	if err != nil {
		panic(fmt.Errorf("error loading contract tree by name and version: %w", err))
	}
	defer rows.Close()

	return scanContractTree(rows)
}

func scanContractTree(rows pgx.Rows) (*model.Contract, bool) {
	var found bool
	var contract *model.Contract

	for rows.Next() {
		var row tableRow

		if err := rows.Scan(
			&row.ContractID,
			&row.ContractVersion,
			&row.ContractRawContract,
			&row.ContractCreatedAt,
			&row.ParticipantID,
			&row.ParticipantName,
			&row.ResourceID,
			&row.ResourceDirection,
			&row.ResourceKind,
			&row.ResourceConsumedProvider,
			&row.ResourceEndpoint,
			&row.ResourceMethod,
			&row.ResourceResponseStatusCode,
			&row.ResourceProviderHash,
			&row.ResourceConsumerHash,
			&row.ResourceCreatedAt,
			&row.PropertyID,
			&row.PropertyPath,
			&row.PropertyVersionType,
			&row.PropertyVersionOptional,
			&row.PropertyVersionChangeType,
		); err != nil {
			panic(fmt.Errorf("error scanning contract tree row: %w", err))
		}

		if row.PropertyVersionChangeType == string(contract_differ.ChangeRemoved) {
			continue
		}

		if !found {
			contract = row.toContractModel()
			found = true
		}

		resource := row.toResourceModel()
		key := resource.PrimaryHash()
		if _, seen := contract.Resources[key]; !seen {
			contract.Resources[key] = resource
		}

		property := row.toPropertyModel()
		if _, seen := contract.Resources[key].Properties[property.Path]; !seen {
			contract.Resources[key].Properties[property.Path] = property
		}
	}

	return contract, found
}

type CurrentConsumerInEnv struct {
	ParticipantID   int64
	ParticipantName string
	Version         string
}

func (r *ContractRepository) FindCurrentConsumersOfProviderInEnv(
	ctx context.Context,
	providerHash string,
	environmentID int64,
) []CurrentConsumerInEnv {
	rows, err := r.pool.Query(ctx, findCurrentConsumersOfProviderInEnvQuery, providerHash, environmentID)
	if err != nil {
		panic(fmt.Errorf("error finding current consumers of provider in env: %w", err))
	}
	defer rows.Close()

	var consumers []CurrentConsumerInEnv
	for rows.Next() {
		var c CurrentConsumerInEnv
		if err := rows.Scan(&c.ParticipantID, &c.ParticipantName, &c.Version); err != nil {
			panic(fmt.Errorf("error scanning current consumer row: %w", err))
		}
		consumers = append(consumers, c)
	}
	return consumers
}

func (r *ContractRepository) LoadProviderResourceOfConsumer(
	ctx context.Context,
	consumer model.Resource,
) (model.Resource, error) {
	rows, err := r.pool.Query(
		ctx,
		loadProviderResourceWithDeploymentsQuery,
		string(model.Provides),
		consumer.ProviderHash(),
	)

	if err != nil {
		panic(fmt.Errorf("error loading provider resource of consumer: %w", err))
	}

	defer rows.Close()

	var found bool
	var provider model.Resource

	for rows.Next() {
		var row tableRow

		if err := rows.Scan(
			&row.ResourceID,
			&row.ParticipantID,
			&row.ParticipantName,
			&row.ResourceDirection,
			&row.ResourceKind,
			&row.ResourceConsumedProvider,
			&row.ResourceEndpoint,
			&row.ResourceMethod,
			&row.ResourceResponseStatusCode,
			&row.ResourceProviderHash,
			&row.ResourceConsumerHash,
			&row.ResourceCreatedAt,
			&row.PropertyPath,
			&row.PropertyVersionType,
			&row.PropertyVersionOptional,
			&row.PropertyVersionChangeType,
			&row.DeploymentEnvironment,
			&row.DeploymentVersion,
		); err != nil {
			panic(fmt.Errorf("error scanning provider resource of consumer: %w", err))
		}

		if !found {
			provider = row.toResourceModel()
			found = true
		}

		provider.Properties[row.PropertyPath] = row.toPropertyModel()

		if row.DeploymentEnvironment.Valid {
			provider.DeployedVersions[row.DeploymentEnvironment.String] = row.DeploymentVersion.String
		}
	}

	if !found {
		return model.Resource{}, ErrProviderResourceNotFound
	}

	return provider, nil
}

func (r *ContractRepository) FindConsumersOfProviderAndEnvironment(
	ctx context.Context,
	provider model.Resource,
	environment *model.Environment,
) []model.Resource {
	rows, err := r.pool.Query(
		ctx,
		findResourcesByDirectionAndProviderHashQuery,
		string(model.Consumes),
		provider.ProviderHash(),
		environment.ID,
	)

	if err != nil {
		panic(fmt.Errorf("error finding consumers of provider: %w", err))
	}

	defer rows.Close()

	consumersMap := make(map[int64]model.Resource)

	for rows.Next() {
		var row tableRow

		if err := rows.Scan(
			&row.ResourceID,
			&row.ParticipantID,
			&row.ParticipantName,
			&row.ResourceDirection,
			&row.ResourceKind,
			&row.ResourceConsumedProvider,
			&row.ResourceEndpoint,
			&row.ResourceMethod,
			&row.ResourceResponseStatusCode,
			&row.ResourceProviderHash,
			&row.ResourceConsumerHash,
			&row.ResourceCreatedAt,
			&row.PropertyPath,
			&row.PropertyVersionType,
			&row.PropertyVersionOptional,
			&row.PropertyVersionChangeType,
			&row.ResourceVersion,
		); err != nil {
			panic(fmt.Errorf("error scanning consumer: %w", err))
		}

		if _, seen := consumersMap[row.ResourceID]; !seen {
			consumersMap[row.ResourceID] = row.toResourceModel()
		}

		consumersMap[row.ResourceID].Properties[row.PropertyPath] = row.toPropertyModel()
	}

	consumers := make([]model.Resource, 0, len(consumersMap))

	for _, consumer := range consumersMap {
		consumers = append(consumers, consumer)
	}

	return consumers
}
