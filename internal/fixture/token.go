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
	agent := InsertAgent(t, ctx, ds, id+"-agent")
	// Insert token directly using SQL
	insertSQL := `
		INSERT INTO api_tokens (id, agent_id, expire_at, created_by, permissions, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, agent_id, expire_at, created_by, permissions, created_at
	`
	var token dbsqlc.ApiToken
	err := ds.QueryRow(ctx, insertSQL,
		id,
		agent.ID,
		expireAt,
		"test",
		permissions,
		createdAt,
	).Scan(
		&token.ID,
		&token.AgentId,
		&token.ExpireAt,
		&token.CreatedBy,
		&token.Permissions,
		&token.CreatedAt,
	)
	if err != nil {
		t.Fatalf("Failed to insert token: %v", err)
	}
	return agent, &token
}
