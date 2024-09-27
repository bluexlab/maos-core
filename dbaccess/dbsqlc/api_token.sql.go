// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: api_token.sql

package dbsqlc

import (
	"context"
)

const apiTokenCount = `-- name: ApiTokenCount :one
SELECT COUNT(*) as count
FROM api_tokens
`

func (q *Queries) ApiTokenCount(ctx context.Context, db DBTX) (int64, error) {
	row := db.QueryRow(ctx, apiTokenCount)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const apiTokenDelete = `-- name: ApiTokenDelete :exec
DELETE FROM api_tokens WHERE id = $1
`

func (q *Queries) ApiTokenDelete(ctx context.Context, db DBTX, id string) error {
	_, err := db.Exec(ctx, apiTokenDelete, id)
	return err
}

const apiTokenFindByID = `-- name: ApiTokenFindByID :one
SELECT t.id, a.id as actor_id, a.queue_id, t.permissions, t.expire_at, t.created_by
FROM api_tokens t
JOIN actors a ON t.actor_id = a.id
WHERE t.id = $1
LIMIT 1
`

type ApiTokenFindByIDRow struct {
	ID          string
	ActorId     int64
	QueueID     int64
	Permissions []string
	ExpireAt    int64
	CreatedBy   string
}

func (q *Queries) ApiTokenFindByID(ctx context.Context, db DBTX, id string) (*ApiTokenFindByIDRow, error) {
	row := db.QueryRow(ctx, apiTokenFindByID, id)
	var i ApiTokenFindByIDRow
	err := row.Scan(
		&i.ID,
		&i.ActorId,
		&i.QueueID,
		&i.Permissions,
		&i.ExpireAt,
		&i.CreatedBy,
	)
	return &i, err
}

const apiTokenInsert = `-- name: ApiTokenInsert :one
INSERT INTO api_tokens(
    id,
    actor_id,
    expire_at,
    created_by,
    permissions,
    created_at
) VALUES (
    $1::text,
    $2::bigint,
    $3::bigint,
    $4::text,
    $5::varchar(255)[],
    EXTRACT(EPOCH FROM NOW())
) RETURNING id, actor_id, expire_at, created_by, created_at, permissions
`

type ApiTokenInsertParams struct {
	ID          string
	ActorId     int64
	ExpireAt    int64
	CreatedBy   string
	Permissions []string
}

func (q *Queries) ApiTokenInsert(ctx context.Context, db DBTX, arg *ApiTokenInsertParams) (*ApiToken, error) {
	row := db.QueryRow(ctx, apiTokenInsert,
		arg.ID,
		arg.ActorId,
		arg.ExpireAt,
		arg.CreatedBy,
		arg.Permissions,
	)
	var i ApiToken
	err := row.Scan(
		&i.ID,
		&i.ActorId,
		&i.ExpireAt,
		&i.CreatedBy,
		&i.CreatedAt,
		&i.Permissions,
	)
	return &i, err
}

const apiTokenListByPage = `-- name: ApiTokenListByPage :many
SELECT
  t.id,
  a.id as actor_id,
  a.name as actor_name,
  a.queue_id,
  t.permissions,
  t.created_at,
  t.expire_at,
  t.created_by,
  COUNT(*) OVER() AS total_count
FROM api_tokens t
JOIN actors a ON t.actor_id = a.id
WHERE ($1::bigint IS NULL OR a.id = $1::bigint)
ORDER BY t.created_at DESC, t.id
LIMIT $2::bigint
OFFSET $2 * ($3::bigint - 1)
`

type ApiTokenListByPageParams struct {
	ActorId  *int64
	PageSize interface{}
	Page     int64
}

type ApiTokenListByPageRow struct {
	ID          string
	ActorId     int64
	ActorName   string
	QueueID     int64
	Permissions []string
	CreatedAt   int64
	ExpireAt    int64
	CreatedBy   string
	TotalCount  int64
}

func (q *Queries) ApiTokenListByPage(ctx context.Context, db DBTX, arg *ApiTokenListByPageParams) ([]*ApiTokenListByPageRow, error) {
	rows, err := db.Query(ctx, apiTokenListByPage, arg.ActorId, arg.PageSize, arg.Page)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*ApiTokenListByPageRow
	for rows.Next() {
		var i ApiTokenListByPageRow
		if err := rows.Scan(
			&i.ID,
			&i.ActorId,
			&i.ActorName,
			&i.QueueID,
			&i.Permissions,
			&i.CreatedAt,
			&i.ExpireAt,
			&i.CreatedBy,
			&i.TotalCount,
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

const apiTokenRotate = `-- name: ApiTokenRotate :one
WITH new_token AS (
  INSERT INTO api_tokens (
    id,
    actor_id,
    expire_at,
    created_by,
    permissions,
    created_at
  ) VALUES (
    $1::text,
    $2::bigint,
    $3::bigint,
    $4::text,
    $5::varchar(255)[],
    EXTRACT(EPOCH FROM NOW())
  )
  RETURNING id
), update_existing AS (
  UPDATE api_tokens
  SET expire_at = EXTRACT(EPOCH FROM NOW() + INTERVAL '15 minutes')
  WHERE actor_id = $2
    AND id != (SELECT id FROM new_token)
)
SELECT id FROM new_token
`

type ApiTokenRotateParams struct {
	ID          string
	ActorId     int64
	NewExpireAt int64
	CreatedBy   string
	Permissions []string
}

func (q *Queries) ApiTokenRotate(ctx context.Context, db DBTX, arg *ApiTokenRotateParams) (string, error) {
	row := db.QueryRow(ctx, apiTokenRotate,
		arg.ID,
		arg.ActorId,
		arg.NewExpireAt,
		arg.CreatedBy,
		arg.Permissions,
	)
	var id string
	err := row.Scan(&id)
	return id, err
}
