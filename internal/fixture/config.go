package fixture

import (
	"context"
	"encoding/json"
	"testing"

	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
)

func InsertConfig(t *testing.T, ctx context.Context, ds DataSource, agentId int64, content map[string]interface{}) *dbsqlc.Config {
	query := dbsqlc.New()
	contentBytes, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("Failed to marshal content: %v", err)
	}
	config, err := query.ConfigInsert(ctx, ds, &dbsqlc.ConfigInsertParams{AgentID: agentId, Content: contentBytes, CreatedBy: "test"})
	if err != nil {
		t.Fatalf("Failed to insert queue: %v", err)
	}

	return config
}
