package fixture

import (
	"context"
	"testing"
	"time"

	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
)

func InsertInvocation(t *testing.T, ctx context.Context, ds DataSource, state string, payload string, actor string) int64 {
	query := dbsqlc.New()
	invocation, err := query.InvocationInsert(ctx, ds, &dbsqlc.InvocationInsertParams{
		State:     dbsqlc.InvocationState(state),
		CreatedAt: time.Now().Unix(),
		Priority:  1,
		Payload:   []byte(payload),
		Metadata:  []byte(`{"kind": "test"}`),
		ActorName: actor,
	})
	if err != nil {
		t.Fatalf("Failed to insert queue: %v", err)
	}

	return invocation.ID
}
