-- name: AgentListPagenated :many
WITH agent_token_count AS (
    SELECT agent_id, COUNT(*) AS token_count
    FROM api_tokens
    GROUP BY agent_id
)
SELECT
  agents.id,
  agents.name,
  agents.queue_id,
  agents.enabled,
  agents.deployable,
  agents.configurable,
  agents.created_at,
  COUNT(*) OVER() AS total_count,
  CASE WHEN atc.token_count IS NULL OR atc.token_count = 0 THEN true ELSE false END AS renameable
FROM agents
LEFT JOIN agent_token_count atc ON agents.id = atc.agent_id
ORDER BY agents.name
LIMIT sqlc.arg(page_size)::bigint
OFFSET sqlc.arg(page_size)::bigint * (sqlc.arg(page)::bigint - 1);

-- name: AgentFindById :one
SELECT *
FROM agents
WHERE id = @id;

-- name: AgentInsert :one
INSERT INTO agents(
    name,
    queue_id,
    enabled,
    deployable,
    configurable,
    metadata
) VALUES (
    @name::text,
    @queue_id::bigint,
    @enabled::boolean,
    @deployable::boolean,
    @configurable::boolean,
    coalesce(@metadata::jsonb, '{}')
) RETURNING *;

-- name: AgentUpdate :one
UPDATE agents SET
    name = COALESCE(sqlc.narg('name')::text, name),
    enabled = COALESCE(sqlc.narg('enabled')::boolean, enabled),
    deployable = COALESCE(sqlc.narg('deployable')::boolean, deployable),
    configurable = COALESCE(sqlc.narg('configurable')::boolean, configurable),
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
