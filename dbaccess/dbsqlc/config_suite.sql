-- name: ConfigSuiteGetById :one
SELECT *
FROM config_suites
WHERE id = @id::bigint
LIMIT 1;

-- name: ConfigSuiteActivate :one
-- Deactivate all other config suites and then activate the given config suite
WITH deactivate_others AS (
    UPDATE config_suites
    SET active = false
    WHERE active = true AND id <> @id::bigint
)
UPDATE config_suites
SET active = true
WHERE id = @id::bigint
RETURNING id;

-- name: ReferenceConfigSuiteList :many
SELECT *
FROM reference_config_suites
ORDER BY name;

-- name: ReferenceConfigSuiteUpsert :one
INSERT INTO reference_config_suites (
  name,
  config_suites
)
VALUES (
  @name::text,
  @config_suites_bytes::jsonb
)
ON CONFLICT (name) DO UPDATE SET
  config_suites = @config_suites_bytes::jsonb,
  updated_at = EXTRACT(EPOCH FROM NOW())
RETURNING id;
