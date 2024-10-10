// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: config.sql

package dbsqlc

import (
	"context"
)

const configActorActiveConfig = `-- name: ConfigActorActiveConfig :one
SELECT configs.id, configs.actor_id, configs.config_suite_id, configs.content, configs.min_actor_version, configs.created_by, configs.created_at, configs.updated_by, configs.updated_at
FROM configs
JOIN actors ON configs.actor_id = actors.id
JOIN config_suites ON configs.config_suite_id = config_suites.id
WHERE configs.actor_id = $1
  AND config_suites.active IS TRUE
  AND config_suites.deployed_at IS NOT NULL
  AND (configs.min_actor_version IS NULL OR $2::integer[] >= configs.min_actor_version::integer[])
ORDER BY configs.id DESC
LIMIT 1
`

type ConfigActorActiveConfigParams struct {
	ActorId      int64
	ActorVersion []int32
}

// Find the active config for the given actor that is compatible with the given actor version
func (q *Queries) ConfigActorActiveConfig(ctx context.Context, db DBTX, arg *ConfigActorActiveConfigParams) (*Config, error) {
	row := db.QueryRow(ctx, configActorActiveConfig, arg.ActorId, arg.ActorVersion)
	var i Config
	err := row.Scan(
		&i.ID,
		&i.ActorId,
		&i.ConfigSuiteID,
		&i.Content,
		&i.MinActorVersion,
		&i.CreatedBy,
		&i.CreatedAt,
		&i.UpdatedBy,
		&i.UpdatedAt,
	)
	return &i, err
}

const configActorRetiredAndVersionCompatibleConfig = `-- name: ConfigActorRetiredAndVersionCompatibleConfig :one
SELECT configs.id, configs.actor_id, configs.config_suite_id, configs.content, configs.min_actor_version, configs.created_by, configs.created_at, configs.updated_by, configs.updated_at
FROM configs
JOIN actors ON configs.actor_id = actors.id
JOIN config_suites ON configs.config_suite_id = config_suites.id
WHERE configs.actor_id = $1
  AND config_suites.active IS FALSE
  AND config_suites.deployed_at IS NOT NULL
  AND (configs.min_actor_version IS NULL OR $2::integer[] >= configs.min_actor_version::integer[])
ORDER BY configs.id DESC
LIMIT 1
`

type ConfigActorRetiredAndVersionCompatibleConfigParams struct {
	ActorId      int64
	ActorVersion []int32
}

// Find the retired config for the given actor that is compatible with the given actor version
func (q *Queries) ConfigActorRetiredAndVersionCompatibleConfig(ctx context.Context, db DBTX, arg *ConfigActorRetiredAndVersionCompatibleConfigParams) (*Config, error) {
	row := db.QueryRow(ctx, configActorRetiredAndVersionCompatibleConfig, arg.ActorId, arg.ActorVersion)
	var i Config
	err := row.Scan(
		&i.ID,
		&i.ActorId,
		&i.ConfigSuiteID,
		&i.Content,
		&i.MinActorVersion,
		&i.CreatedBy,
		&i.CreatedAt,
		&i.UpdatedBy,
		&i.UpdatedAt,
	)
	return &i, err
}

const configFindByActorId = `-- name: ConfigFindByActorId :one
SELECT configs.id, configs.actor_id, configs.config_suite_id, configs.content, configs.min_actor_version, configs.created_by, configs.created_at, configs.updated_by, configs.updated_at, actors.name AS actor_name
FROM configs
JOIN actors ON configs.actor_id = actors.id
WHERE configs.actor_id = $1
ORDER BY configs.created_at DESC, configs.id DESC
LIMIT 1
`

type ConfigFindByActorIdRow struct {
	ID              int64
	ActorId         int64
	ConfigSuiteID   *int64
	Content         []byte
	MinActorVersion []int32
	CreatedBy       string
	CreatedAt       int64
	UpdatedBy       *string
	UpdatedAt       *int64
	ActorName       string
}

func (q *Queries) ConfigFindByActorId(ctx context.Context, db DBTX, actorID int64) (*ConfigFindByActorIdRow, error) {
	row := db.QueryRow(ctx, configFindByActorId, actorID)
	var i ConfigFindByActorIdRow
	err := row.Scan(
		&i.ID,
		&i.ActorId,
		&i.ConfigSuiteID,
		&i.Content,
		&i.MinActorVersion,
		&i.CreatedBy,
		&i.CreatedAt,
		&i.UpdatedBy,
		&i.UpdatedAt,
		&i.ActorName,
	)
	return &i, err
}

const configFindByActorIdAndSuiteId = `-- name: ConfigFindByActorIdAndSuiteId :one
SELECT configs.id, configs.actor_id, configs.config_suite_id, configs.content, configs.min_actor_version, configs.created_by, configs.created_at, configs.updated_by, configs.updated_at, actors.name AS actor_name
FROM configs
JOIN actors ON configs.actor_id = actors.id
WHERE configs.actor_id = $1::bigint
AND configs.config_suite_id = $2::bigint
ORDER BY configs.created_at DESC, configs.id DESC
LIMIT 1
`

type ConfigFindByActorIdAndSuiteIdParams struct {
	ActorId       int64
	ConfigSuiteID int64
}

type ConfigFindByActorIdAndSuiteIdRow struct {
	ID              int64
	ActorId         int64
	ConfigSuiteID   *int64
	Content         []byte
	MinActorVersion []int32
	CreatedBy       string
	CreatedAt       int64
	UpdatedBy       *string
	UpdatedAt       *int64
	ActorName       string
}

func (q *Queries) ConfigFindByActorIdAndSuiteId(ctx context.Context, db DBTX, arg *ConfigFindByActorIdAndSuiteIdParams) (*ConfigFindByActorIdAndSuiteIdRow, error) {
	row := db.QueryRow(ctx, configFindByActorIdAndSuiteId, arg.ActorId, arg.ConfigSuiteID)
	var i ConfigFindByActorIdAndSuiteIdRow
	err := row.Scan(
		&i.ID,
		&i.ActorId,
		&i.ConfigSuiteID,
		&i.Content,
		&i.MinActorVersion,
		&i.CreatedBy,
		&i.CreatedAt,
		&i.UpdatedBy,
		&i.UpdatedAt,
		&i.ActorName,
	)
	return &i, err
}

const configInsert = `-- name: ConfigInsert :one
INSERT INTO configs(
    actor_id,
    content,
    created_by,
    min_actor_version,
    config_suite_id
) VALUES (
    $1::bigint,
    $2::jsonb,
    $3::text,
    $4::integer[],
    $5::bigint
) RETURNING id, actor_id, config_suite_id, content, min_actor_version, created_by, created_at, updated_by, updated_at
`

type ConfigInsertParams struct {
	ActorId         int64
	Content         []byte
	CreatedBy       string
	MinActorVersion []int32
	ConfigSuiteID   *int64
}

func (q *Queries) ConfigInsert(ctx context.Context, db DBTX, arg *ConfigInsertParams) (*Config, error) {
	row := db.QueryRow(ctx, configInsert,
		arg.ActorId,
		arg.Content,
		arg.CreatedBy,
		arg.MinActorVersion,
		arg.ConfigSuiteID,
	)
	var i Config
	err := row.Scan(
		&i.ID,
		&i.ActorId,
		&i.ConfigSuiteID,
		&i.Content,
		&i.MinActorVersion,
		&i.CreatedBy,
		&i.CreatedAt,
		&i.UpdatedBy,
		&i.UpdatedAt,
	)
	return &i, err
}

const configListBySuiteIdGroupByActor = `-- name: ConfigListBySuiteIdGroupByActor :many
SELECT DISTINCT ON (configs.actor_id)
    configs.id,
    configs.actor_id,
    configs.content,
    configs.created_at,
    configs.created_by,
    configs.min_actor_version,
    configs.config_suite_id,
    actors.id AS actor_id,
    actors.name AS actor_name,
    actors.role AS actor_role,
    actors.enabled AS actor_enabled,
    actors.configurable AS actor_configurable,
    actors.deployable AS actor_deployable,
    actors.migratable AS actor_migratable
FROM configs
JOIN actors ON configs.actor_id = actors.id
WHERE configs.config_suite_id = $1::bigint
ORDER BY configs.actor_id, configs.created_at DESC, configs.id DESC
`

type ConfigListBySuiteIdGroupByActorRow struct {
	ID                int64
	ActorId           int64
	Content           []byte
	CreatedAt         int64
	CreatedBy         string
	MinActorVersion   []int32
	ConfigSuiteID     *int64
	ActorId_2         int64
	ActorName         string
	ActorRole         ActorRole
	ActorEnabled      bool
	ActorConfigurable bool
	ActorDeployable   bool
	ActorMigratable   bool
}

func (q *Queries) ConfigListBySuiteIdGroupByActor(ctx context.Context, db DBTX, configSuiteID int64) ([]*ConfigListBySuiteIdGroupByActorRow, error) {
	rows, err := db.Query(ctx, configListBySuiteIdGroupByActor, configSuiteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*ConfigListBySuiteIdGroupByActorRow
	for rows.Next() {
		var i ConfigListBySuiteIdGroupByActorRow
		if err := rows.Scan(
			&i.ID,
			&i.ActorId,
			&i.Content,
			&i.CreatedAt,
			&i.CreatedBy,
			&i.MinActorVersion,
			&i.ConfigSuiteID,
			&i.ActorId_2,
			&i.ActorName,
			&i.ActorRole,
			&i.ActorEnabled,
			&i.ActorConfigurable,
			&i.ActorDeployable,
			&i.ActorMigratable,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const configUpdateInactiveContentByCreator = `-- name: ConfigUpdateInactiveContentByCreator :one
WITH config_suite_check AS (
    SELECT cs.id, d.reviewers
    FROM config_suites cs
    JOIN configs c ON c.config_suite_id = cs.id
    LEFT JOIN deployments d ON d.config_suite_id = cs.id
    WHERE c.id = $4::bigint AND cs.deployed_at IS NULL
)
UPDATE configs SET
    content = COALESCE($1::jsonb, content),
    min_actor_version = COALESCE($2::integer[], min_actor_version),
    updated_at = EXTRACT(EPOCH FROM NOW()),
    updated_by = $3::text
FROM actors
WHERE configs.id = $4::bigint
AND configs.actor_id = actors.id
AND (
    configs.created_by = $3::text
    OR EXISTS (
        SELECT 1 FROM config_suite_check
        WHERE $3::text = ANY(reviewers)
    )
)
AND EXISTS (SELECT 1 FROM config_suite_check)
RETURNING configs.id, configs.actor_id, configs.config_suite_id, configs.content, configs.min_actor_version, configs.created_by, configs.created_at, configs.updated_by, configs.updated_at, actors.name AS actor_name
`

type ConfigUpdateInactiveContentByCreatorParams struct {
	Content         []byte
	MinActorVersion []int32
	Updater         string
	ID              int64
}

type ConfigUpdateInactiveContentByCreatorRow struct {
	ID              int64
	ActorId         int64
	ConfigSuiteID   *int64
	Content         []byte
	MinActorVersion []int32
	CreatedBy       string
	CreatedAt       int64
	UpdatedBy       *string
	UpdatedAt       *int64
	ActorName       string
}

func (q *Queries) ConfigUpdateInactiveContentByCreator(ctx context.Context, db DBTX, arg *ConfigUpdateInactiveContentByCreatorParams) (*ConfigUpdateInactiveContentByCreatorRow, error) {
	row := db.QueryRow(ctx, configUpdateInactiveContentByCreator,
		arg.Content,
		arg.MinActorVersion,
		arg.Updater,
		arg.ID,
	)
	var i ConfigUpdateInactiveContentByCreatorRow
	err := row.Scan(
		&i.ID,
		&i.ActorId,
		&i.ConfigSuiteID,
		&i.Content,
		&i.MinActorVersion,
		&i.CreatedBy,
		&i.CreatedAt,
		&i.UpdatedBy,
		&i.UpdatedAt,
		&i.ActorName,
	)
	return &i, err
}

const getActorByConfigId = `-- name: GetActorByConfigId :one
SELECT actors.id, actors.name, actors.queue_id, actors.created_at, actors.metadata, actors.updated_at, actors.enabled, actors.deployable, actors.configurable, actors.role, actors.migratable
FROM configs
JOIN actors ON configs.actor_id = actors.id
WHERE configs.id = $1::bigint
ORDER BY configs.created_at DESC, configs.id DESC
LIMIT 1
`

func (q *Queries) GetActorByConfigId(ctx context.Context, db DBTX, id int64) (*Actor, error) {
	row := db.QueryRow(ctx, getActorByConfigId, id)
	var i Actor
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.QueueID,
		&i.CreatedAt,
		&i.Metadata,
		&i.UpdatedAt,
		&i.Enabled,
		&i.Deployable,
		&i.Configurable,
		&i.Role,
		&i.Migratable,
	)
	return &i, err
}
