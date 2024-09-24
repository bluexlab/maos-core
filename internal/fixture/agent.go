package fixture

import (
	"context"
	"testing"

	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
)

func InsertAgent(t *testing.T, ctx context.Context, ds DataSource, name string) *dbsqlc.Agent {
	query := dbsqlc.New()
	queue, err := query.QueueInsert(ctx, ds, &dbsqlc.QueueInsertParams{Name: name})
	if err != nil {
		t.Fatalf("Failed to insert queue: %v", err)
	}
	agent, err := query.AgentInsert(ctx, ds, &dbsqlc.AgentInsertParams{
		Name:         name,
		QueueID:      queue.ID,
		Enabled:      true,
		Deployable:   false,
		Configurable: false,
	})
	if err != nil {
		t.Fatalf("Failed to insert agent: %v", err)
	}
	return agent
}

func InsertAgent2(t *testing.T, ctx context.Context, ds DataSource, name string, enabled bool, deployable bool, configurable bool) *dbsqlc.Agent {
	query := dbsqlc.New()
	queue, err := query.QueueInsert(ctx, ds, &dbsqlc.QueueInsertParams{Name: name})
	if err != nil {
		t.Fatalf("Failed to insert queue: %v", err)
	}
	agent, err := query.AgentInsert(ctx, ds, &dbsqlc.AgentInsertParams{
		Name:         name,
		QueueID:      queue.ID,
		Enabled:      enabled,
		Deployable:   deployable,
		Configurable: configurable,
	})
	if err != nil {
		t.Fatalf("Failed to insert agent: %v", err)
	}
	return agent
}
