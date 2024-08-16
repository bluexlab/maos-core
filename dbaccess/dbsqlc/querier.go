// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0

package dbsqlc

import (
	"context"
)

type Querier interface {
	AgentDelete(ctx context.Context, db DBTX, id int64) (string, error)
	AgentFindById(ctx context.Context, db DBTX, id int64) (*Agent, error)
	AgentInsert(ctx context.Context, db DBTX, arg *AgentInsertParams) (*Agent, error)
	AgentListPagenated(ctx context.Context, db DBTX, arg *AgentListPagenatedParams) ([]*AgentListPagenatedRow, error)
	AgentUpdate(ctx context.Context, db DBTX, arg *AgentUpdateParams) (*Agent, error)
	ApiTokenCount(ctx context.Context, db DBTX) (int64, error)
	ApiTokenFindByID(ctx context.Context, db DBTX, id string) (*ApiTokenFindByIDRow, error)
	ApiTokenInsert(ctx context.Context, db DBTX, arg *ApiTokenInsertParams) (*ApiToken, error)
	ApiTokenListByPage(ctx context.Context, db DBTX, arg *ApiTokenListByPageParams) ([]*ApiTokenListByPageRow, error)
	ConfigFindByAgentId(ctx context.Context, db DBTX, agentID int64) ([]*Config, error)
	ConfigInsert(ctx context.Context, db DBTX, arg *ConfigInsertParams) (*Config, error)
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
	TableExists(ctx context.Context, db DBTX, tableName string) (bool, error)
}

var _ Querier = (*Queries)(nil)
