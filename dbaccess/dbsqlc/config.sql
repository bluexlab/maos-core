-- name: ConfigFindByAgentId :one
SELECT configs.*, agents.name AS agent_name
FROM configs
JOIN agents ON configs.agent_id = agents.id
WHERE configs.agent_id = @agent_id
ORDER BY configs.created_at DESC, configs.id DESC
LIMIT 1;

-- name: ConfigFindByAgentIdAndSuiteId :one
SELECT configs.*, agents.name AS agent_name
FROM configs
JOIN agents ON configs.agent_id = agents.id
WHERE configs.agent_id = @agent_id::bigint
AND configs.config_suite_id = @config_suite_id::bigint
ORDER BY configs.created_at DESC, configs.id DESC
LIMIT 1;

-- name: ConfigListBySuiteIdGroupByAgent :many
SELECT DISTINCT ON (configs.agent_id)
    configs.id,
    configs.agent_id,
    configs.content,
    configs.created_at,
    configs.created_by,
    configs.min_agent_version,
    configs.config_suite_id,
    agents.id AS agent_id,
    agents.name AS agent_name
FROM configs
JOIN agents ON configs.agent_id = agents.id
WHERE configs.config_suite_id = @config_suite_id::bigint
ORDER BY configs.agent_id, configs.created_at DESC, configs.id DESC;

-- name: ConfigUpdateInactiveContentByCreator :one
WITH config_suite_check AS (
    SELECT cs.id, d.reviewers
    FROM config_suites cs
    JOIN configs c ON c.config_suite_id = cs.id
    LEFT JOIN deployments d ON d.config_suite_id = cs.id
    WHERE c.id = @id::bigint AND cs.deployed_at IS NULL
)
UPDATE configs SET
    content = COALESCE(sqlc.narg('content')::jsonb, content),
    min_agent_version = COALESCE(sqlc.narg('min_agent_version')::text, min_agent_version),
    updated_at = EXTRACT(EPOCH FROM NOW()),
    updated_by = @updater::text
FROM agents
WHERE configs.id = @id::bigint
AND configs.agent_id = agents.id
AND (
    configs.created_by = @updater::text
    OR EXISTS (
        SELECT 1 FROM config_suite_check
        WHERE @updater::text = ANY(reviewers)
    )
)
AND EXISTS (SELECT 1 FROM config_suite_check)
RETURNING configs.*, agents.name AS agent_name;

-- name: ConfigInsert :one
INSERT INTO configs(
    agent_id,
    content,
    created_by,
    min_agent_version,
    config_suite_id
) VALUES (
    @agent_id::bigint,
    @content::jsonb,
    @created_by::text,
    sqlc.narg('min_agent_version')::text,
    sqlc.narg('config_suite_id')::bigint
) RETURNING *;
