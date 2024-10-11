package apitest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
	"gitlab.com/navyx/ai/maos/maos-core/internal/fixture"
	"gitlab.com/navyx/ai/maos/maos-core/internal/testhelper"
)

func TestAdminTokenCreateEndpoint(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("Valid admin token creation", func(t *testing.T) {
		server, ds, _ := SetupHttpTestWithDb(t, ctx)

		actor := fixture.InsertActor(t, ctx, ds, "actor1")
		fixture.InsertToken(t, ctx, ds, "admin-token", actor.ID, 0, []string{"admin"})

		body := `{"actor_id":1,"created_by":"admin","expire_at":2000000000,"permissions":["config:read","admin"]}`
		resp, resBody := PostHttp(t, server.URL+"/v1/admin/api_tokens", body, "admin-token")
		var token struct {
			ID          string   `json:"id"`
			ActorID     int64    `json:"actor_id"`
			CreatedAt   int64    `json:"created_at"`
			CreatedBy   string   `json:"created_by"`
			ExpireAt    int64    `json:"expire_at"`
			Permissions []string `json:"permissions"`
		}
		err := json.Unmarshal([]byte(resBody), &token)
		require.NoError(t, err)

		require.Equal(t, http.StatusCreated, resp.StatusCode)

		tokens, err := querier.ApiTokenListByPage(ctx, ds, &dbsqlc.ApiTokenListByPageParams{})
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(tokens), 1)

		expectedBody := fmt.Sprintf(`{"actor_id":1, "id":"(ignore)", "created_at":%d, "created_by":"admin", "expire_at":%d, "permissions":["config:read", "admin"]}`, token.CreatedAt, 2000000000)
		testhelper.AssertEqualIgnoringFields(t,
			testhelper.JsonToMap(t, expectedBody),
			testhelper.JsonToMap(t, resBody),
			"id",
		)
	})

	t.Run("Invalid body", func(t *testing.T) {
		server, ds, _ := SetupHttpTestWithDb(t, ctx)

		actor := fixture.InsertActor(t, ctx, ds, "actor1")
		fixture.InsertToken(t, ctx, ds, "admin-token", actor.ID, 0, []string{"admin"})

		body := `{"invalid_json"}`
		resp, resBody := PostHttp(t, server.URL+"/v1/admin/api_tokens", body, "admin-token")

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)

		resJson := testhelper.JsonToMap(t, resBody)
		require.Contains(t, resJson, "error")
		require.Contains(t, resJson["error"], "invalid character ")
	})

	t.Run("Missing required fields", func(t *testing.T) {
		server, ds, _ := SetupHttpTestWithDb(t, ctx)

		actor := fixture.InsertActor(t, ctx, ds, "actor1")
		fixture.InsertToken(t, ctx, ds, "admin-token", actor.ID, 0, []string{"admin"})

		body := `{"actor_id":1}`
		resp, resBody := PostHttp(t, server.URL+"/v1/admin/api_tokens", body, "admin-token")

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)

		resJson := testhelper.JsonToMap(t, resBody)
		require.Contains(t, resJson, "error")
		require.Contains(t, resJson["error"], "Missing required fields")
	})

	t.Run("Non-admin token", func(t *testing.T) {
		server, ds, _ := SetupHttpTestWithDb(t, ctx)

		actor := fixture.InsertActor(t, ctx, ds, "actor1")
		fixture.InsertToken(t, ctx, ds, "actor-token", actor.ID, 0, []string{"user"})

		body := `{"actor_id":1,"created_by":"user","expire_at":2000,"permissions":["config:read"]}`
		resp, _ := PostHttp(t, server.URL+"/v1/admin/api_tokens", body, "actor-token")

		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Invalid token", func(t *testing.T) {
		server, _, _ := SetupHttpTestWithDb(t, ctx)

		body := `{"actor_id":1,"created_by":"admin","expire_at":2000,"permissions":["config:read","admin"]}`
		resp, _ := PostHttp(t, server.URL+"/v1/admin/api_tokens", body, "invalid_token")

		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestDeleteApiToken(t *testing.T) {
	ctx := context.Background()

	t.Run("Successful deletion", func(t *testing.T) {
		server, ds, _ := SetupHttpTestWithDb(t, ctx)

		actor := fixture.InsertActor(t, ctx, ds, "actor1")
		fixture.InsertToken(t, ctx, ds, "admin-token", actor.ID, 0, []string{"admin"})
		tokenToDelete := fixture.InsertToken(t, ctx, ds, "token-to-delete", actor.ID, time.Now().Add(24*time.Hour).Unix(), []string{"read"})

		resp, _ := DeleteHttp(t, fmt.Sprintf("%s/v1/admin/api_tokens/%s", server.URL, tokenToDelete.ID), "admin-token")

		require.Equal(t, http.StatusNoContent, resp.StatusCode)

		// Verify token is deleted
		_, err := querier.ApiTokenFindByID(ctx, ds, tokenToDelete.ID)
		require.Error(t, err)
		require.True(t, errors.Is(err, pgx.ErrNoRows))
	})

	t.Run("Non-existent token", func(t *testing.T) {
		server, ds, _ := SetupHttpTestWithDb(t, ctx)

		actor := fixture.InsertActor(t, ctx, ds, "actor1")
		fixture.InsertToken(t, ctx, ds, "admin-token", actor.ID, 0, []string{"admin"})

		resp, _ := DeleteHttp(t, fmt.Sprintf("%s/v1/admin/api_tokens/non-existent-token", server.URL), "admin-token")

		require.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("Non-admin token", func(t *testing.T) {
		server, ds, _ := SetupHttpTestWithDb(t, ctx)

		actor := fixture.InsertActor(t, ctx, ds, "actor1")
		fixture.InsertToken(t, ctx, ds, "user-token", actor.ID, 0, []string{"user"})
		tokenToDelete := fixture.InsertToken(t, ctx, ds, "token-to-delete", actor.ID, time.Now().Add(24*time.Hour).Unix(), []string{"read"})

		resp, _ := DeleteHttp(t, fmt.Sprintf("%s/v1/admin/api_tokens/%s", server.URL, tokenToDelete.ID), "user-token")

		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Invalid token", func(t *testing.T) {
		server, ds, _ := SetupHttpTestWithDb(t, ctx)

		actor := fixture.InsertActor(t, ctx, ds, "actor1")
		tokenToDelete := fixture.InsertToken(t, ctx, ds, "token-to-delete", actor.ID, time.Now().Add(24*time.Hour).Unix(), []string{"read"})

		resp, _ := DeleteHttp(t, fmt.Sprintf("%s/v1/admin/api_tokens/%s", server.URL, tokenToDelete.ID), "invalid_token")

		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}
