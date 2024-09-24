-- name: PgNotifyOne :exec
WITH topic_to_notify AS (
    SELECT
        concat(current_schema(), '.', @topic::text) AS topic,
        @payload::text AS payload
)
SELECT pg_notify(
    topic_to_notify.topic,
    topic_to_notify.payload
  )
FROM topic_to_notify;
