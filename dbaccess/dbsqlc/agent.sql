-- name: AgentListPagenated :many
SELECT
  id,
  name,
  queue_id,
  created_at,
  COUNT(*) OVER() AS total_count
FROM agents
ORDER BY name
LIMIT sqlc.arg(page_size)::bigint
OFFSET sqlc.arg(page_size) * (sqlc.arg(page)::bigint - 1);

-- name: AgentFindById :one
SELECT *
FROM agents
WHERE id = @id;

-- name: AgentInsert :one
INSERT INTO agents(
    name,
    queue_id,
    metadata
) VALUES (
    @name::text,
    @queue_id::bigint,
    coalesce(@metadata::jsonb, '{}')
) RETURNING *;
