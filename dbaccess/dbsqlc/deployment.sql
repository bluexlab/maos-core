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
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(page_size)::bigint
OFFSET sqlc.arg(page_size) * (sqlc.arg(page)::bigint - 1);

-- name: DeploymentGetById :one
SELECT *
FROM deployments
WHERE id = @id::bigint
LIMIT 1;

-- name: DeploymentGetWithConfigs :one
SELECT deployments.*, configs.*
FROM deployments
LEFT JOIN configs ON deployments.id = configs.deployment_id
WHERE deployments.id = @id::bigint
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
  sqlc.arg(created_by)::text
)
RETURNING *;

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

-- name: DeploymentDelete :one
DELETE FROM deployments
WHERE id = @id::bigint AND status = 'draft'
RETURNING *;
