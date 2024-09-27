-- name: ActorListPagenated :many
WITH actor_token_count AS (
  SELECT actor_id, COUNT(*) AS token_count
  FROM api_tokens
  GROUP BY actor_id
)
SELECT
  actors.id,
  actors.name,
  actors.role,
  actors.queue_id,
  actors.enabled,
  actors.deployable,
  actors.configurable,
  actors.created_at,
  COUNT(*) OVER() AS total_count,
  COALESCE(atc.token_count, 0) AS token_count,
  CASE WHEN atc.token_count IS NULL OR atc.token_count = 0 THEN true ELSE false END AS renameable
FROM actors
LEFT JOIN actor_token_count atc ON actors.id = atc.actor_id
ORDER BY actors.name
LIMIT sqlc.arg(page_size)::bigint
OFFSET sqlc.arg(page_size)::bigint * (sqlc.arg(page)::bigint - 1);

-- name: ActorFindById :one
WITH actor_token_count AS (
  SELECT actor_id, COUNT(*) AS token_count
  FROM api_tokens
  WHERE actor_id = @id
  GROUP BY actor_id
)
SELECT
  actors.id,
  actors.name,
  actors.queue_id,
  actors.role,
  actors.enabled,
  actors.deployable,
  actors.configurable,
  actors.created_at,
  COALESCE(atc.token_count, 0) AS token_count,
  CASE WHEN atc.token_count IS NULL OR atc.token_count = 0 THEN true ELSE false END AS renameable
FROM actors
LEFT JOIN actor_token_count atc ON actors.id = atc.actor_id
WHERE actors.id = @id;

-- name: ActorInsert :one
INSERT INTO actors(
    name,
    queue_id,
    role,
    enabled,
    deployable,
    configurable,
    metadata
) VALUES (
    @name::text,
    @queue_id::bigint,
    @role::actor_role,
    @enabled::boolean,
    @deployable::boolean,
    @configurable::boolean,
    coalesce(@metadata::jsonb, '{}')
) RETURNING *;

-- name: ActorUpdate :one
UPDATE actors SET
    name = COALESCE(sqlc.narg('name')::text, name),
    role = COALESCE(sqlc.narg('role')::actor_role, role),
    enabled = COALESCE(sqlc.narg('enabled')::boolean, enabled),
    deployable = COALESCE(sqlc.narg('deployable')::boolean, deployable),
    configurable = COALESCE(sqlc.narg('configurable')::boolean, configurable),
    metadata = COALESCE(sqlc.narg('metadata')::jsonb, metadata)
WHERE id = @id
RETURNING *;

-- name: ActorDelete :one
WITH check_actor AS (
    SELECT EXISTS (SELECT 1 FROM actors WHERE actors.id = @id) AS actor_exists
),
check_config AS (
    SELECT EXISTS (SELECT 1 FROM configs WHERE configs.actor_id = @id) AS config_exists
),
delete_actor AS (
    DELETE FROM actors
    WHERE actors.id = @id
    AND EXISTS (SELECT 1 FROM check_actor WHERE actor_exists = true)
    AND NOT EXISTS (SELECT 1 FROM check_config WHERE config_exists = true)
    RETURNING *
)
SELECT
    CASE
        WHEN NOT EXISTS (SELECT 1 FROM check_actor WHERE actor_exists = true) THEN 'NOTFOUND'
        WHEN EXISTS (SELECT 1 FROM check_config WHERE config_exists = true) THEN 'REFERENCED'
        WHEN EXISTS (SELECT 1 FROM delete_actor) THEN 'DONE'
        ELSE 'ERROR'
    END AS result;
