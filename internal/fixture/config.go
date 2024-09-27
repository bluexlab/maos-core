package fixture

import (
	"context"
	"encoding/json"
	"testing"

	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
)

func InsertConfig(t *testing.T, ctx context.Context, ds DataSource, actorId int64, content map[string]string) *dbsqlc.Config {
	query := dbsqlc.New()
	contentBytes, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("Failed to marshal content: %v", err)
	}
	config, err := query.ConfigInsert(ctx, ds, &dbsqlc.ConfigInsertParams{ActorId: actorId, Content: contentBytes, CreatedBy: "test"})
	if err != nil {
		t.Fatalf("Failed to insert config: %v", err)
	}

	return config
}

func InsertConfig2(t *testing.T, ctx context.Context, ds DataSource, actorId int64, configSuiteId *int64, createdBy string, content map[string]string) *dbsqlc.Config {
	query := dbsqlc.New()
	contentBytes, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("Failed to marshal content: %v", err)
	}
	config, err := query.ConfigInsert(
		ctx,
		ds,
		&dbsqlc.ConfigInsertParams{
			ActorId:       actorId,
			ConfigSuiteID: configSuiteId,
			Content:       contentBytes,
			CreatedBy:     createdBy,
		},
	)
	if err != nil {
		t.Fatalf("Failed to insert config: %v", err)
	}

	return config
}

func InsertConfigSuite(t *testing.T, ctx context.Context, ds DataSource) *dbsqlc.ConfigSuite {
	row := ds.QueryRow(ctx, "INSERT INTO config_suites (created_by) VALUES ('tester') RETURNING id, active, created_by, deployed_at")

	var configSuite dbsqlc.ConfigSuite
	err := row.Scan(&configSuite.ID, &configSuite.Active, &configSuite.CreatedBy, &configSuite.DeployedAt)
	if err != nil {
		t.Fatalf("Failed to scan config suite: %v", err)
	}
	return &configSuite
}
