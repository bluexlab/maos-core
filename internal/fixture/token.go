package fixture

import (
	"context"
	"testing"

	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
)

func InsertToken(t *testing.T, ctx context.Context, ds DataSource, id string, agentId int64, expireAt int64, permissions []string) *dbsqlc.ApiToken {
	query := dbsqlc.New()
	token, err := query.ApiTokenInsert(ctx, ds, &dbsqlc.ApiTokenInsertParams{
		ID:          id,
		AgentId:     agentId,
		ExpireAt:    expireAt,
		CreatedBy:   "test",
		Permissions: permissions,
	})
	if err != nil {
		t.Fatalf("Failed to insert agent: %v", err)
	}
	return token
}

func InsertAgentToken(t *testing.T, ctx context.Context, ds DataSource, id string, expireAt int64, permissions []string, createdAt int64) (*dbsqlc.Agent, *dbsqlc.ApiToken) {
	query := dbsqlc.New()
	agent := InsertAgent(t, ctx, ds, id+"-agent")
	token, err := query.ApiTokenInsert(ctx, ds, &dbsqlc.ApiTokenInsertParams{
		ID:          id,
		AgentId:     agent.ID,
		ExpireAt:    expireAt,
		CreatedBy:   "test",
		Permissions: permissions,
		CreatedAt:   createdAt,
	})
	if err != nil {
		t.Fatalf("Failed to insert agent: %v", err)
	}
	return agent, token
}
