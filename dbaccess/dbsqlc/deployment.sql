-- name: DeploymentListPaginated :many
SELECT
  id,
  name,
  status,
  reviewers,
  notes,
  created_at,
  created_by,
  approved_at,
  approved_by,
  finished_at,
  finished_by,
  COUNT(*) OVER() AS total_count
FROM deployments
WHERE (sqlc.narg(reviewer)::text IS NULL OR sqlc.narg(reviewer)::text = ANY(reviewers))
  AND (sqlc.narg(status)::deployment_status IS NULL OR status = sqlc.narg(status)::deployment_status)
  AND (sqlc.narg(name)::text IS NULL OR name ILIKE '%' || sqlc.narg(name)::text || '%')
  AND (sqlc.narg(id)::bigint[] IS NULL OR id = ANY(sqlc.narg(id)::bigint[]))
ORDER BY status, created_at DESC, id DESC
LIMIT sqlc.arg(page_size)::bigint
OFFSET sqlc.arg(page_size) * (sqlc.arg(page)::bigint - 1);

-- name: DeploymentGetById :one
SELECT *
FROM deployments
WHERE id = @id::bigint
LIMIT 1;

-- name: DeploymentInsert :one
INSERT INTO deployments (
  name,
  status,
  reviewers,
  created_by
)
VALUES (
  sqlc.arg(name)::text,
  COALESCE(sqlc.narg(status)::deployment_status, 'draft'),
  COALESCE(sqlc.narg(reviewers)::text[], '{}'),
  @created_by::text
)
RETURNING *;

-- name: DeploymentInsertWithConfigSuite :one
-- Create a new deployment with an associated config suite.
-- For each actor:
--   1. If there is an active config suite, duplicate the config from the active config suite.
--   2. If there is no active config suite, duplicate the latest config from the actor.
--   3. If the actor has no existing config, create a new config with default values.
-- Associate all these new configs with the newly created deployment and config suite.
WITH inserted_config_suite AS (
  INSERT INTO config_suites (created_by)
  VALUES (@created_by::text)
  RETURNING id
),
inserted_deployment AS (
  INSERT INTO deployments (
    name,
    status,
    reviewers,
    created_by,
    config_suite_id
  )
  VALUES (
    sqlc.arg(name)::text,
    'draft',
    COALESCE(sqlc.narg(reviewers)::text[], '{}'),
    @created_by::text,
    (SELECT id FROM inserted_config_suite)
  )
  RETURNING *
),
active_config_suites AS (
  SELECT id FROM config_suites WHERE active = TRUE
),
actor_configs AS (
  INSERT INTO configs (actor_id, config_suite_id, created_by, min_actor_version, content)
  SELECT
    actors.id,
    (SELECT id FROM inserted_config_suite),
    @created_by::text,
    COALESCE(
      (SELECT min_actor_version FROM configs WHERE actor_id = actors.id ORDER BY created_at DESC LIMIT 1),
      NULL
    ),
    COALESCE(
      (SELECT content FROM configs
        WHERE actor_id = actors.id
          AND config_suite_id IN (SELECT id FROM active_config_suites)
        ORDER BY created_at DESC, id DESC
        LIMIT 1),
      (SELECT content FROM configs
        WHERE actor_id = actors.id
        ORDER BY created_at DESC, id DESC
        LIMIT 1),
      '{}'::jsonb
    )
  FROM actors
  WHERE configurable = TRUE
)
SELECT * FROM inserted_deployment;

-- name: DeploymentCloneFrom :one
-- Clone a deployment and its associated config suite.
-- The new deployment will be in the draft status.
WITH source_deployment AS (
  SELECT config_suite_id FROM deployments WHERE id = @clone_from::bigint
),
inserted_config_suite AS (
  INSERT INTO config_suites (created_by)
  SELECT @created_by::text
  FROM source_deployment
  RETURNING id
),
inserted_deployment AS (
  INSERT INTO deployments (
    name,
    status,
    reviewers,
    created_by,
    config_suite_id
  )
  SELECT
    sqlc.arg(name)::text,
    'draft',
    COALESCE(sqlc.narg(reviewers)::text[], '{}'),
    @created_by::text,
    (SELECT id FROM inserted_config_suite)
  FROM source_deployment
  RETURNING *
),
actor_configs AS (
  INSERT INTO configs (actor_id, config_suite_id, created_by, min_actor_version, content)
  SELECT
    actors.id,
    (SELECT config_suite_id FROM inserted_deployment),
    @created_by::text,
    COALESCE(
      (SELECT min_actor_version FROM configs WHERE actor_id = actors.id ORDER BY created_at DESC LIMIT 1),
      NULL
    ),
    COALESCE(
      (SELECT content FROM configs
        WHERE actor_id = actors.id
          AND config_suite_id = (SELECT config_suite_id FROM source_deployment)
        ORDER BY created_at DESC, id DESC
        LIMIT 1),
      (SELECT content FROM configs
        WHERE actor_id = actors.id
        ORDER BY created_at DESC, id DESC
        LIMIT 1),
      '{}'::jsonb
    )
  FROM actors
  WHERE configurable = TRUE AND EXISTS (SELECT 1 FROM source_deployment)
)
SELECT * FROM inserted_deployment;


-- name: DeploymentUpdate :one
UPDATE deployments
SET
  name = COALESCE(sqlc.narg(name)::text, name),
  reviewers = COALESCE(sqlc.narg(reviewers)::text[], reviewers)
WHERE id = @id::bigint AND status = 'draft'
RETURNING *;

-- name: DeploymentSubmitForReview :one
UPDATE deployments
SET status = 'reviewing'
WHERE id = @id::bigint AND status = 'draft'
RETURNING *;

-- name: SetDeploymentDeploying :one
-- it sets the specific deployment status to deploying.
-- it checks if the deployment status is in draft or reviewing before setting it to deploying
-- it also checks if there are no other deploying deployments
WITH deployment_info AS (
  SELECT d.status,
         EXISTS (
           SELECT 1 FROM deployments WHERE status = 'deploying'
         ) AS others_deploying
  FROM deployments d
  WHERE d.id = @id::bigint
),
update_result AS (
  UPDATE deployments
  SET status = 'deploying',
    deploying_at = EXTRACT(EPOCH FROM NOW()),
    approved_at = EXTRACT(EPOCH FROM NOW()),
    approved_by = @approved_by::text
  FROM deployment_info
  WHERE deployments.id = @id::bigint
    AND deployment_info.status IN ('reviewing', 'draft')
    AND NOT deployment_info.others_deploying
  RETURNING *
)
SELECT
  CASE
    WHEN d.status NOT IN ('reviewing', 'draft') THEN
      'deployment must be in reviewing or draft status'::text
    WHEN d.others_deploying THEN
      'others are deploying'::text
    ELSE
      ''::text
  END AS result
FROM deployment_info AS d
UNION ALL
SELECT 'deployment not found' AS result
WHERE NOT EXISTS (SELECT 1 FROM deployment_info);

-- name: DeploymentPublish :one
-- it sets current deployed deployment status to retired and the new deployment status to deployed
WITH deactivate_others AS (
  UPDATE deployments
  SET status = 'retired',
    finished_at = EXTRACT(EPOCH FROM NOW()),
    finished_by = @approved_by::text
  WHERE status = 'deployed'
  RETURNING id
)
UPDATE deployments
SET status = 'deployed',
  deployed_at = EXTRACT(EPOCH FROM NOW())
WHERE id = @id::bigint
  AND id NOT IN (SELECT id FROM deactivate_others)
  AND status = 'deploying'
RETURNING *;

-- name: DeploymentReject :one
-- Reject a deployment.
-- The deployment must be in the reviewing status and the user must be a reviewer.
UPDATE deployments
SET status = 'rejected',
  finished_at = EXTRACT(EPOCH FROM NOW()),
  finished_by = @rejected_by::text,
  notes = @notes::jsonb
WHERE id = @id::bigint
  AND status = 'reviewing'
  AND @rejected_by::text = ANY(reviewers)
RETURNING *;

-- name: UpdateDeploymentMigrationLogs :exec
UPDATE deployments
SET migration_logs = @migration_logs::jsonb
WHERE id = @id::bigint;

-- name: UpdateDeploymentLastError :exec
UPDATE deployments
SET last_error = @last_error::text, status = 'failed'
WHERE id = @id::bigint;

-- name: DeploymentDelete :one
DELETE FROM deployments
WHERE id = @id::bigint AND status = 'draft'
RETURNING *;
