// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: config.sql

package dbsqlc

import (
	"context"
)

const configFindByAgentId = `-- name: ConfigFindByAgentId :one
SELECT configs.id, configs.agent_id, configs.config_suite_id, configs.content, configs.min_agent_version, configs.created_by, configs.created_at, configs.updated_by, configs.updated_at, agents.name AS agent_name
FROM configs
JOIN agents ON configs.agent_id = agents.id
WHERE configs.agent_id = $1
ORDER BY configs.created_at DESC, configs.id DESC
LIMIT 1
`

type ConfigFindByAgentIdRow struct {
	ID              int64
	AgentId         int64
	ConfigSuiteID   *int64
	Content         []byte
	MinAgentVersion *string
	CreatedBy       string
	CreatedAt       int64
	UpdatedBy       *string
	UpdatedAt       *int64
	AgentName       string
}

func (q *Queries) ConfigFindByAgentId(ctx context.Context, db DBTX, agentID int64) (*ConfigFindByAgentIdRow, error) {
	row := db.QueryRow(ctx, configFindByAgentId, agentID)
	var i ConfigFindByAgentIdRow
	err := row.Scan(
		&i.ID,
		&i.AgentId,
		&i.ConfigSuiteID,
		&i.Content,
		&i.MinAgentVersion,
		&i.CreatedBy,
		&i.CreatedAt,
		&i.UpdatedBy,
		&i.UpdatedAt,
		&i.AgentName,
	)
	return &i, err
}

const configFindByAgentIdAndSuiteId = `-- name: ConfigFindByAgentIdAndSuiteId :one
SELECT configs.id, configs.agent_id, configs.config_suite_id, configs.content, configs.min_agent_version, configs.created_by, configs.created_at, configs.updated_by, configs.updated_at, agents.name AS agent_name
FROM configs
JOIN agents ON configs.agent_id = agents.id
WHERE configs.agent_id = $1::bigint
AND configs.config_suite_id = $2::bigint
ORDER BY configs.created_at DESC, configs.id DESC
LIMIT 1
`

type ConfigFindByAgentIdAndSuiteIdParams struct {
	AgentId       int64
	ConfigSuiteID int64
}

type ConfigFindByAgentIdAndSuiteIdRow struct {
	ID              int64
	AgentId         int64
	ConfigSuiteID   *int64
	Content         []byte
	MinAgentVersion *string
	CreatedBy       string
	CreatedAt       int64
	UpdatedBy       *string
	UpdatedAt       *int64
	AgentName       string
}

func (q *Queries) ConfigFindByAgentIdAndSuiteId(ctx context.Context, db DBTX, arg *ConfigFindByAgentIdAndSuiteIdParams) (*ConfigFindByAgentIdAndSuiteIdRow, error) {
	row := db.QueryRow(ctx, configFindByAgentIdAndSuiteId, arg.AgentId, arg.ConfigSuiteID)
	var i ConfigFindByAgentIdAndSuiteIdRow
	err := row.Scan(
		&i.ID,
		&i.AgentId,
		&i.ConfigSuiteID,
		&i.Content,
		&i.MinAgentVersion,
		&i.CreatedBy,
		&i.CreatedAt,
		&i.UpdatedBy,
		&i.UpdatedAt,
		&i.AgentName,
	)
	return &i, err
}

const configInsert = `-- name: ConfigInsert :one
INSERT INTO configs(
    agent_id,
    content,
    created_by,
    min_agent_version,
    config_suite_id
) VALUES (
    $1::bigint,
    $2::jsonb,
    $3::text,
    $4::text,
    $5::bigint
) RETURNING id, agent_id, config_suite_id, content, min_agent_version, created_by, created_at, updated_by, updated_at
`

type ConfigInsertParams struct {
	AgentId         int64
	Content         []byte
	CreatedBy       string
	MinAgentVersion *string
	ConfigSuiteID   *int64
}

func (q *Queries) ConfigInsert(ctx context.Context, db DBTX, arg *ConfigInsertParams) (*Config, error) {
	row := db.QueryRow(ctx, configInsert,
		arg.AgentId,
		arg.Content,
		arg.CreatedBy,
		arg.MinAgentVersion,
		arg.ConfigSuiteID,
	)
	var i Config
	err := row.Scan(
		&i.ID,
		&i.AgentId,
		&i.ConfigSuiteID,
		&i.Content,
		&i.MinAgentVersion,
		&i.CreatedBy,
		&i.CreatedAt,
		&i.UpdatedBy,
		&i.UpdatedAt,
	)
	return &i, err
}

const configListBySuiteIdGroupByAgent = `-- name: ConfigListBySuiteIdGroupByAgent :many
SELECT DISTINCT ON (configs.agent_id)
    configs.id,
    configs.agent_id,
    configs.content,
    configs.created_at,
    configs.created_by,
    configs.min_agent_version,
    configs.config_suite_id,
    agents.id AS agent_id,
    agents.name AS agent_name
FROM configs
JOIN agents ON configs.agent_id = agents.id
WHERE configs.config_suite_id = $1::bigint
ORDER BY configs.agent_id, configs.created_at DESC, configs.id DESC
`

type ConfigListBySuiteIdGroupByAgentRow struct {
	ID              int64
	AgentId         int64
	Content         []byte
	CreatedAt       int64
	CreatedBy       string
	MinAgentVersion *string
	ConfigSuiteID   *int64
	AgentId_2       int64
	AgentName       string
}

func (q *Queries) ConfigListBySuiteIdGroupByAgent(ctx context.Context, db DBTX, configSuiteID int64) ([]*ConfigListBySuiteIdGroupByAgentRow, error) {
	rows, err := db.Query(ctx, configListBySuiteIdGroupByAgent, configSuiteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*ConfigListBySuiteIdGroupByAgentRow
	for rows.Next() {
		var i ConfigListBySuiteIdGroupByAgentRow
		if err := rows.Scan(
			&i.ID,
			&i.AgentId,
			&i.Content,
			&i.CreatedAt,
			&i.CreatedBy,
			&i.MinAgentVersion,
			&i.ConfigSuiteID,
			&i.AgentId_2,
			&i.AgentName,
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
    min_agent_version = COALESCE($2::text, min_agent_version),
    updated_at = EXTRACT(EPOCH FROM NOW()),
    updated_by = $3::text
FROM agents
WHERE configs.id = $4::bigint
AND configs.agent_id = agents.id
AND (
    configs.created_by = $3::text
    OR EXISTS (
        SELECT 1 FROM config_suite_check
        WHERE $3::text = ANY(reviewers)
    )
)
AND EXISTS (SELECT 1 FROM config_suite_check)
RETURNING configs.id, configs.agent_id, configs.config_suite_id, configs.content, configs.min_agent_version, configs.created_by, configs.created_at, configs.updated_by, configs.updated_at, agents.name AS agent_name
`

type ConfigUpdateInactiveContentByCreatorParams struct {
	Content         []byte
	MinAgentVersion *string
	Updater         string
	ID              int64
}

type ConfigUpdateInactiveContentByCreatorRow struct {
	ID              int64
	AgentId         int64
	ConfigSuiteID   *int64
	Content         []byte
	MinAgentVersion *string
	CreatedBy       string
	CreatedAt       int64
	UpdatedBy       *string
	UpdatedAt       *int64
	AgentName       string
}

func (q *Queries) ConfigUpdateInactiveContentByCreator(ctx context.Context, db DBTX, arg *ConfigUpdateInactiveContentByCreatorParams) (*ConfigUpdateInactiveContentByCreatorRow, error) {
	row := db.QueryRow(ctx, configUpdateInactiveContentByCreator,
		arg.Content,
		arg.MinAgentVersion,
		arg.Updater,
		arg.ID,
	)
	var i ConfigUpdateInactiveContentByCreatorRow
	err := row.Scan(
		&i.ID,
		&i.AgentId,
		&i.ConfigSuiteID,
		&i.Content,
		&i.MinAgentVersion,
		&i.CreatedBy,
		&i.CreatedAt,
		&i.UpdatedBy,
		&i.UpdatedAt,
		&i.AgentName,
	)
	return &i, err
}
