package fixture

import (
	"context"
	"testing"

	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
)

func InsertAgent(t *testing.T, ctx context.Context, ds DataSource, name string) *dbsqlc.Agents {
	query := dbsqlc.New()
	queue, err := query.QueueInsert(ctx, ds, &dbsqlc.QueueInsertParams{Name: name})
	if err != nil {
		t.Fatalf("Failed to insert queue: %v", err)
	}
	agent, err := query.AgentInsert(ctx, ds, &dbsqlc.AgentInsertParams{Name: name, QueueID: queue.ID})
	if err != nil {
		t.Fatalf("Failed to insert agent: %v", err)
	}
	return agent
}
