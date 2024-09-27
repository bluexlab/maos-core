package fixture

import (
	"context"
	"testing"

	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
)

func InsertToken(t *testing.T, ctx context.Context, ds DataSource, id string, actorId int64, expireAt int64, permissions []string) *dbsqlc.ApiToken {
	query := dbsqlc.New()
	token, err := query.ApiTokenInsert(ctx, ds, &dbsqlc.ApiTokenInsertParams{
		ID:          id,
		ActorId:     actorId,
		ExpireAt:    expireAt,
		CreatedBy:   "test",
		Permissions: permissions,
	})
	if err != nil {
		t.Fatalf("Failed to insert actor: %v", err)
	}
	return token
}

func InsertActorToken(t *testing.T, ctx context.Context, ds DataSource, id string, expireAt int64, permissions []string, createdAt int64) (*dbsqlc.Actor, *dbsqlc.ApiToken) {
	actor := InsertActor(t, ctx, ds, id+"-actor")
	// Insert token directly using SQL
	insertSQL := `
		INSERT INTO api_tokens (id, actor_id, expire_at, created_by, permissions, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, actor_id, expire_at, created_by, permissions, created_at
	`
	var token dbsqlc.ApiToken
	err := ds.QueryRow(ctx, insertSQL,
		id,
		actor.ID,
		expireAt,
		"test",
		permissions,
		createdAt,
	).Scan(
		&token.ID,
		&token.ActorId,
		&token.ExpireAt,
		&token.CreatedBy,
		&token.Permissions,
		&token.CreatedAt,
	)
	if err != nil {
		t.Fatalf("Failed to insert token: %v", err)
	}
	return actor, &token
}
