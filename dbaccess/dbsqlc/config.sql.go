// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: config.sql

package dbsqlc

import (
	"context"
)

const configFindByAgentId = `-- name: ConfigFindByAgentId :many
SELECT id, agent_id, config_suite_id, content, min_agent_version, created_by, created_at, updated_by, updated_at
FROM configs
WHERE agent_id = $1
ORDER BY created_at DESC
`

func (q *Queries) ConfigFindByAgentId(ctx context.Context, db DBTX, agentID int64) ([]*Config, error) {
	rows, err := db.Query(ctx, configFindByAgentId, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*Config
	for rows.Next() {
		var i Config
		if err := rows.Scan(
			&i.ID,
			&i.AgentID,
			&i.ConfigSuiteID,
			&i.Content,
			&i.MinAgentVersion,
			&i.CreatedBy,
			&i.CreatedAt,
			&i.UpdatedBy,
			&i.UpdatedAt,
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

const configInsert = `-- name: ConfigInsert :one
INSERT INTO configs(
    agent_id,
    content,
    created_by,
    min_agent_version
) VALUES (
    $1::bigint,
    $2::jsonb,
    $3::text,
    $4::text
) RETURNING id, agent_id, config_suite_id, content, min_agent_version, created_by, created_at, updated_by, updated_at
`

type ConfigInsertParams struct {
	AgentID         int64
	Content         []byte
	CreatedBy       string
	MinAgentVersion *string
}

func (q *Queries) ConfigInsert(ctx context.Context, db DBTX, arg *ConfigInsertParams) (*Config, error) {
	row := db.QueryRow(ctx, configInsert,
		arg.AgentID,
		arg.Content,
		arg.CreatedBy,
		arg.MinAgentVersion,
	)
	var i Config
	err := row.Scan(
		&i.ID,
		&i.AgentID,
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
