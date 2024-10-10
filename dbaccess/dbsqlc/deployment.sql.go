// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: deployment.sql

package dbsqlc

import (
	"context"
)

const deploymentDelete = `-- name: DeploymentDelete :one
DELETE FROM deployments
WHERE id = $1::bigint AND status = 'draft'
RETURNING id, name, status, reviewers, config_suite_id, notes, created_by, created_at, approved_by, approved_at, finished_by, finished_at, migration_logs, last_error, deploying_at, deployed_at
`

func (q *Queries) DeploymentDelete(ctx context.Context, db DBTX, id int64) (*Deployment, error) {
	row := db.QueryRow(ctx, deploymentDelete, id)
	var i Deployment
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Status,
		&i.Reviewers,
		&i.ConfigSuiteID,
		&i.Notes,
		&i.CreatedBy,
		&i.CreatedAt,
		&i.ApprovedBy,
		&i.ApprovedAt,
		&i.FinishedBy,
		&i.FinishedAt,
		&i.MigrationLogs,
		&i.LastError,
		&i.DeployingAt,
		&i.DeployedAt,
	)
	return &i, err
}

const deploymentGetById = `-- name: DeploymentGetById :one
SELECT id, name, status, reviewers, config_suite_id, notes, created_by, created_at, approved_by, approved_at, finished_by, finished_at, migration_logs, last_error, deploying_at, deployed_at
FROM deployments
WHERE id = $1::bigint
LIMIT 1
`

func (q *Queries) DeploymentGetById(ctx context.Context, db DBTX, id int64) (*Deployment, error) {
	row := db.QueryRow(ctx, deploymentGetById, id)
	var i Deployment
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Status,
		&i.Reviewers,
		&i.ConfigSuiteID,
		&i.Notes,
		&i.CreatedBy,
		&i.CreatedAt,
		&i.ApprovedBy,
		&i.ApprovedAt,
		&i.FinishedBy,
		&i.FinishedAt,
		&i.MigrationLogs,
		&i.LastError,
		&i.DeployingAt,
		&i.DeployedAt,
	)
	return &i, err
}

const deploymentInsert = `-- name: DeploymentInsert :one
INSERT INTO deployments (
  name,
  status,
  reviewers,
  created_by
)
VALUES (
  $1::text,
  COALESCE($2::deployment_status, 'draft'),
  COALESCE($3::text[], '{}'),
  $4::text
)
RETURNING id, name, status, reviewers, config_suite_id, notes, created_by, created_at, approved_by, approved_at, finished_by, finished_at, migration_logs, last_error, deploying_at, deployed_at
`

type DeploymentInsertParams struct {
	Name      string
	Status    NullDeploymentStatus
	Reviewers []string
	CreatedBy string
}

func (q *Queries) DeploymentInsert(ctx context.Context, db DBTX, arg *DeploymentInsertParams) (*Deployment, error) {
	row := db.QueryRow(ctx, deploymentInsert,
		arg.Name,
		arg.Status,
		arg.Reviewers,
		arg.CreatedBy,
	)
	var i Deployment
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Status,
		&i.Reviewers,
		&i.ConfigSuiteID,
		&i.Notes,
		&i.CreatedBy,
		&i.CreatedAt,
		&i.ApprovedBy,
		&i.ApprovedAt,
		&i.FinishedBy,
		&i.FinishedAt,
		&i.MigrationLogs,
		&i.LastError,
		&i.DeployingAt,
		&i.DeployedAt,
	)
	return &i, err
}

const deploymentInsertWithConfigSuite = `-- name: DeploymentInsertWithConfigSuite :one
WITH inserted_config_suite AS (
  INSERT INTO config_suites (created_by)
  VALUES ($1::text)
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
    $2::text,
    'draft',
    COALESCE($3::text[], '{}'),
    $1::text,
    (SELECT id FROM inserted_config_suite)
  )
  RETURNING id, name, status, reviewers, config_suite_id, notes, created_by, created_at, approved_by, approved_at, finished_by, finished_at, migration_logs, last_error, deploying_at, deployed_at
),
actor_configs AS (
  INSERT INTO configs (actor_id, config_suite_id, created_by, min_actor_version, content)
  SELECT
    actors.id,
    (SELECT id FROM inserted_config_suite),
    $1::text,
    COALESCE(
      (SELECT min_actor_version FROM configs WHERE actor_id = actors.id ORDER BY created_at DESC LIMIT 1),
      NULL
    ),
    COALESCE(
      (SELECT content FROM configs WHERE actor_id = actors.id ORDER BY created_at DESC LIMIT 1),
      '{}'::jsonb
    )
  FROM actors
)
SELECT id, name, status, reviewers, config_suite_id, notes, created_by, created_at, approved_by, approved_at, finished_by, finished_at, migration_logs, last_error, deploying_at, deployed_at FROM inserted_deployment
`

type DeploymentInsertWithConfigSuiteParams struct {
	CreatedBy string
	Name      string
	Reviewers []string
}

type DeploymentInsertWithConfigSuiteRow struct {
	ID            int64
	Name          string
	Status        DeploymentStatus
	Reviewers     []string
	ConfigSuiteID *int64
	Notes         []byte
	CreatedBy     string
	CreatedAt     int64
	ApprovedBy    *string
	ApprovedAt    *int64
	FinishedBy    *string
	FinishedAt    *int64
	MigrationLogs []byte
	LastError     *string
	DeployingAt   *int64
	DeployedAt    *int64
}

// Create a new deployment with an associated config suite.
// For each actor:
//  1. If the actor has an existing config, duplicate its latest config.
//  2. If the actor has no existing config, create a new config with default values.
//
// Associate all these new configs with the newly created deployment and config suite.
func (q *Queries) DeploymentInsertWithConfigSuite(ctx context.Context, db DBTX, arg *DeploymentInsertWithConfigSuiteParams) (*DeploymentInsertWithConfigSuiteRow, error) {
	row := db.QueryRow(ctx, deploymentInsertWithConfigSuite, arg.CreatedBy, arg.Name, arg.Reviewers)
	var i DeploymentInsertWithConfigSuiteRow
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Status,
		&i.Reviewers,
		&i.ConfigSuiteID,
		&i.Notes,
		&i.CreatedBy,
		&i.CreatedAt,
		&i.ApprovedBy,
		&i.ApprovedAt,
		&i.FinishedBy,
		&i.FinishedAt,
		&i.MigrationLogs,
		&i.LastError,
		&i.DeployingAt,
		&i.DeployedAt,
	)
	return &i, err
}

const deploymentListPaginated = `-- name: DeploymentListPaginated :many
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
WHERE ($1::text IS NULL OR $1::text = ANY(reviewers))
  AND ($2::deployment_status IS NULL OR status = $2::deployment_status)
  AND ($3::text IS NULL OR name ILIKE '%' || $3::text || '%')
  AND ($4::bigint[] IS NULL OR id = ANY($4::bigint[]))
ORDER BY status, created_at DESC, id DESC
LIMIT $5::bigint
OFFSET $5 * ($6::bigint - 1)
`

type DeploymentListPaginatedParams struct {
	Reviewer *string
	Status   NullDeploymentStatus
	Name     *string
	ID       []int64
	PageSize interface{}
	Page     int64
}

type DeploymentListPaginatedRow struct {
	ID         int64
	Name       string
	Status     DeploymentStatus
	Reviewers  []string
	Notes      []byte
	CreatedAt  int64
	CreatedBy  string
	ApprovedAt *int64
	ApprovedBy *string
	FinishedAt *int64
	FinishedBy *string
	TotalCount int64
}

func (q *Queries) DeploymentListPaginated(ctx context.Context, db DBTX, arg *DeploymentListPaginatedParams) ([]*DeploymentListPaginatedRow, error) {
	rows, err := db.Query(ctx, deploymentListPaginated,
		arg.Reviewer,
		arg.Status,
		arg.Name,
		arg.ID,
		arg.PageSize,
		arg.Page,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*DeploymentListPaginatedRow
	for rows.Next() {
		var i DeploymentListPaginatedRow
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Status,
			&i.Reviewers,
			&i.Notes,
			&i.CreatedAt,
			&i.CreatedBy,
			&i.ApprovedAt,
			&i.ApprovedBy,
			&i.FinishedAt,
			&i.FinishedBy,
			&i.TotalCount,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const deploymentPublish = `-- name: DeploymentPublish :one
WITH deactivate_others AS (
  UPDATE deployments
  SET status = 'retired',
    finished_at = EXTRACT(EPOCH FROM NOW()),
    finished_by = $2::text
  WHERE status = 'deployed'
  RETURNING id
)
UPDATE deployments
SET status = 'deployed',
  deployed_at = EXTRACT(EPOCH FROM NOW())
WHERE id = $1::bigint
  AND id NOT IN (SELECT id FROM deactivate_others)
  AND status = 'deploying'
RETURNING id, name, status, reviewers, config_suite_id, notes, created_by, created_at, approved_by, approved_at, finished_by, finished_at, migration_logs, last_error, deploying_at, deployed_at
`

type DeploymentPublishParams struct {
	ID         int64
	ApprovedBy string
}

// it sets current deployed deployment status to retired and the new deployment status to deployed
func (q *Queries) DeploymentPublish(ctx context.Context, db DBTX, arg *DeploymentPublishParams) (*Deployment, error) {
	row := db.QueryRow(ctx, deploymentPublish, arg.ID, arg.ApprovedBy)
	var i Deployment
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Status,
		&i.Reviewers,
		&i.ConfigSuiteID,
		&i.Notes,
		&i.CreatedBy,
		&i.CreatedAt,
		&i.ApprovedBy,
		&i.ApprovedAt,
		&i.FinishedBy,
		&i.FinishedAt,
		&i.MigrationLogs,
		&i.LastError,
		&i.DeployingAt,
		&i.DeployedAt,
	)
	return &i, err
}

const deploymentReject = `-- name: DeploymentReject :one
UPDATE deployments
SET status = 'rejected',
  finished_at = EXTRACT(EPOCH FROM NOW()),
  finished_by = $1::text,
  notes = $2::jsonb
WHERE id = $3::bigint
  AND status = 'reviewing'
  AND $1::text = ANY(reviewers)
RETURNING id, name, status, reviewers, config_suite_id, notes, created_by, created_at, approved_by, approved_at, finished_by, finished_at, migration_logs, last_error, deploying_at, deployed_at
`

type DeploymentRejectParams struct {
	RejectedBy string
	Notes      []byte
	ID         int64
}

// Reject a deployment.
// The deployment must be in the reviewing status and the user must be a reviewer.
func (q *Queries) DeploymentReject(ctx context.Context, db DBTX, arg *DeploymentRejectParams) (*Deployment, error) {
	row := db.QueryRow(ctx, deploymentReject, arg.RejectedBy, arg.Notes, arg.ID)
	var i Deployment
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Status,
		&i.Reviewers,
		&i.ConfigSuiteID,
		&i.Notes,
		&i.CreatedBy,
		&i.CreatedAt,
		&i.ApprovedBy,
		&i.ApprovedAt,
		&i.FinishedBy,
		&i.FinishedAt,
		&i.MigrationLogs,
		&i.LastError,
		&i.DeployingAt,
		&i.DeployedAt,
	)
	return &i, err
}

const deploymentSubmitForReview = `-- name: DeploymentSubmitForReview :one
UPDATE deployments
SET status = 'reviewing'
WHERE id = $1::bigint AND status = 'draft'
RETURNING id, name, status, reviewers, config_suite_id, notes, created_by, created_at, approved_by, approved_at, finished_by, finished_at, migration_logs, last_error, deploying_at, deployed_at
`

func (q *Queries) DeploymentSubmitForReview(ctx context.Context, db DBTX, id int64) (*Deployment, error) {
	row := db.QueryRow(ctx, deploymentSubmitForReview, id)
	var i Deployment
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Status,
		&i.Reviewers,
		&i.ConfigSuiteID,
		&i.Notes,
		&i.CreatedBy,
		&i.CreatedAt,
		&i.ApprovedBy,
		&i.ApprovedAt,
		&i.FinishedBy,
		&i.FinishedAt,
		&i.MigrationLogs,
		&i.LastError,
		&i.DeployingAt,
		&i.DeployedAt,
	)
	return &i, err
}

const deploymentUpdate = `-- name: DeploymentUpdate :one
UPDATE deployments
SET
  name = COALESCE($1::text, name),
  reviewers = COALESCE($2::text[], reviewers)
WHERE id = $3::bigint AND status = 'draft'
RETURNING id, name, status, reviewers, config_suite_id, notes, created_by, created_at, approved_by, approved_at, finished_by, finished_at, migration_logs, last_error, deploying_at, deployed_at
`

type DeploymentUpdateParams struct {
	Name      *string
	Reviewers []string
	ID        int64
}

func (q *Queries) DeploymentUpdate(ctx context.Context, db DBTX, arg *DeploymentUpdateParams) (*Deployment, error) {
	row := db.QueryRow(ctx, deploymentUpdate, arg.Name, arg.Reviewers, arg.ID)
	var i Deployment
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Status,
		&i.Reviewers,
		&i.ConfigSuiteID,
		&i.Notes,
		&i.CreatedBy,
		&i.CreatedAt,
		&i.ApprovedBy,
		&i.ApprovedAt,
		&i.FinishedBy,
		&i.FinishedAt,
		&i.MigrationLogs,
		&i.LastError,
		&i.DeployingAt,
		&i.DeployedAt,
	)
	return &i, err
}

const setDeploymentDeploying = `-- name: SetDeploymentDeploying :one
WITH deployment_info AS (
  SELECT d.status,
         EXISTS (
           SELECT 1 FROM deployments WHERE status = 'deploying'
         ) AS others_deploying
  FROM deployments d
  WHERE d.id = $1::bigint
),
update_result AS (
  UPDATE deployments
  SET status = 'deploying',
    deploying_at = EXTRACT(EPOCH FROM NOW()),
    approved_at = EXTRACT(EPOCH FROM NOW()),
    approved_by = $2::text
  FROM deployment_info
  WHERE deployments.id = $1::bigint
    AND deployment_info.status IN ('reviewing', 'draft')
    AND NOT deployment_info.others_deploying
  RETURNING deployment_info.status, others_deploying, id, name, deployments.status, reviewers, config_suite_id, notes, created_by, created_at, approved_by, approved_at, finished_by, finished_at, migration_logs, last_error, deploying_at, deployed_at
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
WHERE NOT EXISTS (SELECT 1 FROM deployment_info)
`

type SetDeploymentDeployingParams struct {
	ID         int64
	ApprovedBy string
}

// it sets the specific deployment status to deploying.
// it checks if the deployment status is in draft or reviewing before setting it to deploying
// it also checks if there are no other deploying deployments
func (q *Queries) SetDeploymentDeploying(ctx context.Context, db DBTX, arg *SetDeploymentDeployingParams) (string, error) {
	row := db.QueryRow(ctx, setDeploymentDeploying, arg.ID, arg.ApprovedBy)
	var result string
	err := row.Scan(&result)
	return result, err
}

const updateDeploymentLastError = `-- name: UpdateDeploymentLastError :exec
UPDATE deployments
SET last_error = $1::text, status = 'failed'
WHERE id = $2::bigint
`

type UpdateDeploymentLastErrorParams struct {
	LastError string
	ID        int64
}

func (q *Queries) UpdateDeploymentLastError(ctx context.Context, db DBTX, arg *UpdateDeploymentLastErrorParams) error {
	_, err := db.Exec(ctx, updateDeploymentLastError, arg.LastError, arg.ID)
	return err
}

const updateDeploymentMigrationLogs = `-- name: UpdateDeploymentMigrationLogs :exec
UPDATE deployments
SET migration_logs = $1::jsonb
WHERE id = $2::bigint
`

type UpdateDeploymentMigrationLogsParams struct {
	MigrationLogs []byte
	ID            int64
}

func (q *Queries) UpdateDeploymentMigrationLogs(ctx context.Context, db DBTX, arg *UpdateDeploymentMigrationLogsParams) error {
	_, err := db.Exec(ctx, updateDeploymentMigrationLogs, arg.MigrationLogs, arg.ID)
	return err
}
