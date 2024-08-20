-- name: DeploymentListPaginated :many
SELECT
  id,
  name,
  status,
  created_at,
  created_by,
  approved_at,
  approved_by,
  finished_at,
  finished_by,
  COUNT(*) OVER() AS total_count
FROM deployments
ORDER BY status,created_at DESC, id DESC
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
-- For each agent:
--   1. If the agent has an existing config, duplicate its latest config.
--   2. If the agent has no existing config, create a new config with default values.
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
agent_configs AS (
  INSERT INTO configs (agent_id, config_suite_id, created_by, min_agent_version, content)
  SELECT
    agents.id,
    (SELECT id FROM inserted_config_suite),
    @created_by::text,
    COALESCE(
      (SELECT min_agent_version FROM configs WHERE agent_id = agents.id ORDER BY created_at DESC LIMIT 1),
      NULL
    ),
    COALESCE(
      (SELECT content FROM configs WHERE agent_id = agents.id ORDER BY created_at DESC LIMIT 1),
      '{}'::jsonb
    )
  FROM agents
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

-- name: DeploymentPublish :one
-- it sets current deployed deployment status to retired and the new deployment status to deployed
WITH current_deployed AS (
  UPDATE deployments
  SET status = 'retired',
    finished_at = EXTRACT(EPOCH FROM NOW()),
    finished_by = @approved_by::text
  WHERE status = 'deployed'
  RETURNING id
)
UPDATE deployments
SET status = 'deployed',
approved_at = EXTRACT(EPOCH FROM NOW()),
approved_by = @approved_by::text
WHERE id = @id::bigint AND (status = 'reviewing' OR status = 'draft')
RETURNING *;


-- name: DeploymentDelete :one
DELETE FROM deployments
WHERE id = @id::bigint AND status = 'draft'
RETURNING *;
