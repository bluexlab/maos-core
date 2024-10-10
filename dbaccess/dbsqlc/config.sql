-- name: ConfigFindByActorId :one
SELECT configs.*, actors.name AS actor_name
FROM configs
JOIN actors ON configs.actor_id = actors.id
WHERE configs.actor_id = @actor_id
ORDER BY configs.created_at DESC, configs.id DESC
LIMIT 1;

-- name: ConfigActorActiveConfig :one
-- Find the active config for the given actor that is compatible with the given actor version
SELECT configs.*
FROM configs
JOIN actors ON configs.actor_id = actors.id
JOIN config_suites ON configs.config_suite_id = config_suites.id
WHERE configs.actor_id = @actor_id
  AND config_suites.active IS TRUE
  AND config_suites.deployed_at IS NOT NULL
  AND (configs.min_actor_version IS NULL OR @actor_version::integer[] >= configs.min_actor_version::integer[])
ORDER BY configs.id DESC
LIMIT 1;

-- name: ConfigActorRetiredAndVersionCompatibleConfig :one
-- Find the retired config for the given actor that is compatible with the given actor version
SELECT configs.*
FROM configs
JOIN actors ON configs.actor_id = actors.id
JOIN config_suites ON configs.config_suite_id = config_suites.id
WHERE configs.actor_id = @actor_id
  AND config_suites.active IS FALSE
  AND config_suites.deployed_at IS NOT NULL
  AND (configs.min_actor_version IS NULL OR @actor_version::integer[] >= configs.min_actor_version::integer[])
ORDER BY configs.id DESC
LIMIT 1;

-- name: ConfigFindByActorIdAndSuiteId :one
SELECT configs.*, actors.name AS actor_name
FROM configs
JOIN actors ON configs.actor_id = actors.id
WHERE configs.actor_id = @actor_id::bigint
AND configs.config_suite_id = @config_suite_id::bigint
ORDER BY configs.created_at DESC, configs.id DESC
LIMIT 1;

-- name: GetActorByConfigId :one
SELECT actors.*
FROM configs
JOIN actors ON configs.actor_id = actors.id
WHERE configs.id = @id::bigint
ORDER BY configs.created_at DESC, configs.id DESC
LIMIT 1;

-- name: ConfigListBySuiteIdGroupByActor :many
SELECT DISTINCT ON (configs.actor_id)
    configs.id,
    configs.actor_id,
    configs.content,
    configs.created_at,
    configs.created_by,
    configs.min_actor_version,
    configs.config_suite_id,
    actors.id AS actor_id,
    actors.name AS actor_name,
    actors.role AS actor_role,
    actors.enabled AS actor_enabled,
    actors.configurable AS actor_configurable,
    actors.deployable AS actor_deployable,
    actors.migratable AS actor_migratable
FROM configs
JOIN actors ON configs.actor_id = actors.id
WHERE configs.config_suite_id = @config_suite_id::bigint
ORDER BY configs.actor_id, configs.created_at DESC, configs.id DESC;

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
    min_actor_version = COALESCE(sqlc.narg('min_actor_version')::integer[], min_actor_version),
    updated_at = EXTRACT(EPOCH FROM NOW()),
    updated_by = @updater::text
FROM actors
WHERE configs.id = @id::bigint
AND configs.actor_id = actors.id
AND (
    configs.created_by = @updater::text
    OR EXISTS (
        SELECT 1 FROM config_suite_check
        WHERE @updater::text = ANY(reviewers)
    )
)
AND EXISTS (SELECT 1 FROM config_suite_check)
RETURNING configs.*, actors.name AS actor_name;

-- name: ConfigInsert :one
INSERT INTO configs(
    actor_id,
    content,
    created_by,
    min_actor_version,
    config_suite_id
) VALUES (
    @actor_id::bigint,
    @content::jsonb,
    @created_by::text,
    sqlc.narg('min_actor_version')::integer[],
    sqlc.narg('config_suite_id')::bigint
) RETURNING *;
