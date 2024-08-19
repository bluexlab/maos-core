-- name: ConfigFindByAgentId :one
SELECT configs.*, agents.name AS agent_name
FROM configs
JOIN agents ON configs.agent_id = agents.id
WHERE configs.agent_id = @agent_id
ORDER BY configs.created_at DESC, configs.id DESC
LIMIT 1;

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
