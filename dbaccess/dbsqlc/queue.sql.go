// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: queue.sql

package dbsqlc

import (
	"context"
)

const queueFindById = `-- name: QueueFindById :one
SELECT id, name, created_at, metadata, paused_at, updated_at
FROM queues
WHERE id = $1
`

func (q *Queries) QueueFindById(ctx context.Context, db DBTX, id int64) (*Queue, error) {
	row := db.QueryRow(ctx, queueFindById, id)
	var i Queue
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.CreatedAt,
		&i.Metadata,
		&i.PausedAt,
		&i.UpdatedAt,
	)
	return &i, err
}

const queueInsert = `-- name: QueueInsert :one
INSERT INTO queues(
    name,
    metadata
) VALUES (
    $1::text,
    coalesce($2::jsonb, '{}')
) RETURNING id, name, created_at, metadata, paused_at, updated_at
`

type QueueInsertParams struct {
	Name     string
	Metadata []byte
}

func (q *Queries) QueueInsert(ctx context.Context, db DBTX, arg *QueueInsertParams) (*Queue, error) {
	row := db.QueryRow(ctx, queueInsert, arg.Name, arg.Metadata)
	var i Queue
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.CreatedAt,
		&i.Metadata,
		&i.PausedAt,
		&i.UpdatedAt,
	)
	return &i, err
}
