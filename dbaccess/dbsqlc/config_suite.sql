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
