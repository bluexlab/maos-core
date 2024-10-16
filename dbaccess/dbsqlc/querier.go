// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0

package dbsqlc

import (
	"context"
)

type Querier interface {
	ActorDelete(ctx context.Context, db DBTX, id int64) (string, error)
	ActorFindById(ctx context.Context, db DBTX, id int64) (*ActorFindByIdRow, error)
	ActorInsert(ctx context.Context, db DBTX, arg *ActorInsertParams) (*Actor, error)
	ActorListPagenated(ctx context.Context, db DBTX, arg *ActorListPagenatedParams) ([]*ActorListPagenatedRow, error)
	ActorUpdate(ctx context.Context, db DBTX, arg *ActorUpdateParams) (*Actor, error)
	ApiTokenCount(ctx context.Context, db DBTX) (int64, error)
	ApiTokenDelete(ctx context.Context, db DBTX, id string) error
	ApiTokenFindByID(ctx context.Context, db DBTX, id string) (*ApiTokenFindByIDRow, error)
	ApiTokenInsert(ctx context.Context, db DBTX, arg *ApiTokenInsertParams) (*ApiToken, error)
	ApiTokenListByPage(ctx context.Context, db DBTX, arg *ApiTokenListByPageParams) ([]*ApiTokenListByPageRow, error)
	ApiTokenRotate(ctx context.Context, db DBTX, arg *ApiTokenRotateParams) (string, error)
	// Find the active config for the given actor that is compatible with the given actor version
	ConfigActorActiveConfig(ctx context.Context, db DBTX, arg *ConfigActorActiveConfigParams) (*Config, error)
	// Find the retired config for the given actor that is compatible with the given actor version
	ConfigActorRetiredAndVersionCompatibleConfig(ctx context.Context, db DBTX, arg *ConfigActorRetiredAndVersionCompatibleConfigParams) (*Config, error)
	ConfigFindByActorId(ctx context.Context, db DBTX, actorID int64) (*ConfigFindByActorIdRow, error)
	ConfigFindByActorIdAndSuiteId(ctx context.Context, db DBTX, arg *ConfigFindByActorIdAndSuiteIdParams) (*ConfigFindByActorIdAndSuiteIdRow, error)
	ConfigInsert(ctx context.Context, db DBTX, arg *ConfigInsertParams) (*Config, error)
	ConfigListBySuiteIdGroupByActor(ctx context.Context, db DBTX, configSuiteID int64) ([]*ConfigListBySuiteIdGroupByActorRow, error)
	// Deactivate all other config suites and then activate the given config suite
	ConfigSuiteActivate(ctx context.Context, db DBTX, arg *ConfigSuiteActivateParams) (int64, error)
	ConfigSuiteGetById(ctx context.Context, db DBTX, id int64) (*ConfigSuite, error)
	ConfigUpdateInactiveContentByCreator(ctx context.Context, db DBTX, arg *ConfigUpdateInactiveContentByCreatorParams) (*ConfigUpdateInactiveContentByCreatorRow, error)
	// Clone a deployment and its associated config suite.
	// The new deployment will be in the draft status.
	DeploymentCloneFrom(ctx context.Context, db DBTX, arg *DeploymentCloneFromParams) (*DeploymentCloneFromRow, error)
	DeploymentDelete(ctx context.Context, db DBTX, id int64) (*Deployment, error)
	DeploymentGetById(ctx context.Context, db DBTX, id int64) (*Deployment, error)
	DeploymentInsert(ctx context.Context, db DBTX, arg *DeploymentInsertParams) (*Deployment, error)
	// Create a new deployment with an associated config suite.
	// For each actor:
	//   1. If there is an active config suite, duplicate the config from the active config suite.
	//   2. If there is no active config suite, duplicate the latest config from the actor.
	//   3. If the actor has no existing config, create a new config with default values.
	// Associate all these new configs with the newly created deployment and config suite.
	DeploymentInsertWithConfigSuite(ctx context.Context, db DBTX, arg *DeploymentInsertWithConfigSuiteParams) (*DeploymentInsertWithConfigSuiteRow, error)
	DeploymentListPaginated(ctx context.Context, db DBTX, arg *DeploymentListPaginatedParams) ([]*DeploymentListPaginatedRow, error)
	// it sets current deployed deployment status to retired and the new deployment status to deployed
	DeploymentPublish(ctx context.Context, db DBTX, arg *DeploymentPublishParams) (*Deployment, error)
	// Reject a deployment.
	// The deployment must be in the reviewing status and the user must be a reviewer.
	DeploymentReject(ctx context.Context, db DBTX, arg *DeploymentRejectParams) (*Deployment, error)
	DeploymentSubmitForReview(ctx context.Context, db DBTX, id int64) (*Deployment, error)
	DeploymentUpdate(ctx context.Context, db DBTX, arg *DeploymentUpdateParams) (*Deployment, error)
	GetActorByConfigId(ctx context.Context, db DBTX, id int64) (*Actor, error)
	InvocationFindById(ctx context.Context, db DBTX, id int64) (*Invocation, error)
	InvocationGetAvailable(ctx context.Context, db DBTX, arg *InvocationGetAvailableParams) ([]*Invocation, error)
	InvocationInsert(ctx context.Context, db DBTX, arg *InvocationInsertParams) (*InvocationInsertRow, error)
	InvocationSetCompleteIfRunning(ctx context.Context, db DBTX, arg *InvocationSetCompleteIfRunningParams) (*InvocationSetCompleteIfRunningRow, error)
	InvocationSetFailureIfRunning(ctx context.Context, db DBTX, arg *InvocationSetFailureIfRunningParams) (*InvocationSetFailureIfRunningRow, error)
	MigrationDeleteByVersionMany(ctx context.Context, db DBTX, version []int64) ([]*Migration, error)
	MigrationGetAll(ctx context.Context, db DBTX) ([]*Migration, error)
	MigrationInsert(ctx context.Context, db DBTX, version int64) (*Migration, error)
	MigrationInsertMany(ctx context.Context, db DBTX, version []int64) ([]*Migration, error)
	PgNotifyOne(ctx context.Context, db DBTX, arg *PgNotifyOneParams) error
	QueueFindById(ctx context.Context, db DBTX, id int64) (*Queue, error)
	QueueInsert(ctx context.Context, db DBTX, arg *QueueInsertParams) (*Queue, error)
	ReferenceConfigSuiteList(ctx context.Context, db DBTX) ([]*ReferenceConfigSuites, error)
	ReferenceConfigSuiteUpsert(ctx context.Context, db DBTX, arg *ReferenceConfigSuiteUpsertParams) (int64, error)
	// it sets the specific deployment status to deploying.
	// it checks if the deployment status is in draft or reviewing before setting it to deploying
	// it also checks if there are no other deploying deployments
	SetDeploymentDeploying(ctx context.Context, db DBTX, arg *SetDeploymentDeployingParams) (string, error)
	SettingGetSystem(ctx context.Context, db DBTX) (*Settings, error)
	SettingUpdateSystem(ctx context.Context, db DBTX, value []byte) (*Settings, error)
	TableExists(ctx context.Context, db DBTX, tableName string) (bool, error)
	UpdateDeploymentLastError(ctx context.Context, db DBTX, arg *UpdateDeploymentLastErrorParams) error
	UpdateDeploymentMigrationLogs(ctx context.Context, db DBTX, arg *UpdateDeploymentMigrationLogsParams) error
}

var _ Querier = (*Queries)(nil)
