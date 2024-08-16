-- name: ConfigFindByAgentId :many
SELECT *
FROM configs
WHERE agent_id = @agent_id
ORDER BY created_at DESC;

-- name: ConfigInsert :one
INSERT INTO configs(
    agent_id,
    content,
    created_by,
    min_agent_version
) VALUES (
    @agent_id::bigint,
    @content::jsonb,
    @created_by::text,
    sqlc.narg('min_agent_version')::text
) RETURNING *;

