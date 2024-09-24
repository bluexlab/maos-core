package fixture

import (
	"context"
	"testing"

	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
)

func InsertDeployment(t *testing.T, ctx context.Context, ds DataSource, name string, reviewers []string) *dbsqlc.Deployment {
	query := dbsqlc.New()
	deployment, err := query.DeploymentInsert(ctx, ds, &dbsqlc.DeploymentInsertParams{Name: name, Reviewers: reviewers, CreatedBy: "tester"})
	if err != nil {
		t.Fatalf("Failed to insert deployment: %v", err)
	}
	return deployment
}
