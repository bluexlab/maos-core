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

-- name: AgentUpdate :one
UPDATE agents SET
    name = COALESCE(sqlc.narg('name')::text, name),
    metadata = COALESCE(sqlc.narg('metadata')::jsonb, metadata)
WHERE id = @id
RETURNING *;

-- name: AgentDelete :one
WITH check_agent AS (
    SELECT EXISTS (SELECT 1 FROM agents WHERE agents.id = @id) AS agent_exists
),
check_config AS (
    SELECT EXISTS (SELECT 1 FROM configs WHERE configs.agent_id = @id) AS config_exists
),
delete_agent AS (
    DELETE FROM agents
    WHERE agents.id = @id
    AND EXISTS (SELECT 1 FROM check_agent WHERE agent_exists = true)
    AND NOT EXISTS (SELECT 1 FROM check_config WHERE config_exists = true)
    RETURNING *
)
SELECT
    CASE
        WHEN NOT EXISTS (SELECT 1 FROM check_agent WHERE agent_exists = true) THEN 'NOTFOUND'
        WHEN EXISTS (SELECT 1 FROM check_config WHERE config_exists = true) THEN 'REFERENCED'
        WHEN EXISTS (SELECT 1 FROM delete_agent) THEN 'DONE'
        ELSE 'ERROR'
    END AS result;
