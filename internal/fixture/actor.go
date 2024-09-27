package fixture

import (
	"context"
	"testing"

	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
)

func InsertActor(t *testing.T, ctx context.Context, ds DataSource, name string) *dbsqlc.Actor {
	query := dbsqlc.New()
	queue, err := query.QueueInsert(ctx, ds, &dbsqlc.QueueInsertParams{Name: name})
	if err != nil {
		t.Fatalf("Failed to insert queue: %v", err)
	}
	actor, err := query.ActorInsert(ctx, ds, &dbsqlc.ActorInsertParams{
		Name:         name,
		Role:         dbsqlc.ActorRole("agent"),
		QueueID:      queue.ID,
		Enabled:      true,
		Deployable:   false,
		Configurable: false,
	})
	if err != nil {
		t.Fatalf("Failed to insert actor: %v", err)
	}
	return actor
}

func InsertActor2(t *testing.T, ctx context.Context, ds DataSource, name string, role string, enabled bool, deployable bool, configurable bool) *dbsqlc.Actor {
	query := dbsqlc.New()
	queue, err := query.QueueInsert(ctx, ds, &dbsqlc.QueueInsertParams{Name: name})
	if err != nil {
		t.Fatalf("Failed to insert queue: %v", err)
	}
	actor, err := query.ActorInsert(ctx, ds, &dbsqlc.ActorInsertParams{
		Name:         name,
		Role:         dbsqlc.ActorRole(role),
		QueueID:      queue.ID,
		Enabled:      enabled,
		Deployable:   deployable,
		Configurable: configurable,
	})
	if err != nil {
		t.Fatalf("Failed to insert actor: %v", err)
	}
	return actor
}
