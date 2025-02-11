// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: notify.sql

package dbsqlc

import (
	"context"
)

const pgNotifyOne = `-- name: PgNotifyOne :exec
WITH topic_to_notify AS (
    SELECT
        concat(current_schema(), '.', $1::text) AS topic,
        $2::text AS payload
)
SELECT pg_notify(
    topic_to_notify.topic,
    topic_to_notify.payload
  )
FROM topic_to_notify
`

type PgNotifyOneParams struct {
	Topic   string
	Payload string
}

func (q *Queries) PgNotifyOne(ctx context.Context, db DBTX, arg *PgNotifyOneParams) error {
	_, err := db.Exec(ctx, pgNotifyOne, arg.Topic, arg.Payload)
	return err
}
