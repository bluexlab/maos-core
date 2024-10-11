package apitest

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/internal/fixture"
	"gitlab.com/navyx/ai/maos/maos-core/internal/testhelper"
)

func TestAdminListActorEndpoint(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	server, ds, _ := SetupHttpTestWithDb(t, ctx)

	actor1 := fixture.InsertActor(t, ctx, ds, "actor1")
	actor2 := fixture.InsertActor(t, ctx, ds, "actor2")
	fixture.InsertToken(t, ctx, ds, "admin-token", actor1.ID, 0, []string{"admin"})
	fixture.InsertToken(t, ctx, ds, "actor-token", actor2.ID, 0, []string{"user"})

	t.Run("Valid admin token", func(t *testing.T) {
		resp, resBody := GetHttp(t, server.URL+"/v1/admin/actors", "admin-token")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var response api.AdminListActors200JSONResponse
		err := json.Unmarshal([]byte(resBody), &response)
		require.NoError(t, err)

		require.Len(t, response.Data, 2)
		require.Equal(t, 1, response.Meta.TotalPages)

		for _, actor := range response.Data {
			require.NotZero(t, actor.Id)
			require.NotEmpty(t, actor.Name)
			require.NotZero(t, actor.CreatedAt)
		}
	})

	t.Run("Valid admin token with pagination", func(t *testing.T) {
		resp, resBody := GetHttp(t, server.URL+"/v1/admin/actors?page=1&page_size=1", "admin-token")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var response api.AdminListActors200JSONResponse
		err := json.Unmarshal([]byte(resBody), &response)
		require.NoError(t, err)

		require.Len(t, response.Data, 1)
		require.Equal(t, 2, response.Meta.TotalPages)
	})

	t.Run("Non-admin token", func(t *testing.T) {
		resp, _ := GetHttp(t, server.URL+"/v1/admin/actors", "actor-token")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Invalid token", func(t *testing.T) {
		resp, _ := GetHttp(t, server.URL+"/v1/admin/actors", "invalid_token")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestAdminCreateActorEndpoint(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	server, ds, _ := SetupHttpTestWithDb(t, ctx)

	actor1 := fixture.InsertActor(t, ctx, ds, "actor1")
	actor2 := fixture.InsertActor(t, ctx, ds, "actor2")
	fixture.InsertToken(t, ctx, ds, "admin-token", actor1.ID, 0, []string{"admin"})
	fixture.InsertToken(t, ctx, ds, "actor-token", actor2.ID, 0, []string{"user"})

	t.Run("Valid admin token", func(t *testing.T) {
		resp, resBody := PostHttp(t, server.URL+"/v1/admin/actors", `{"name":"new_actor","role":"portal"}`, "admin-token")
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		var response api.AdminCreateActor201JSONResponse
		err := json.Unmarshal([]byte(resBody), &response)
		require.NoError(t, err)

		require.NotZero(t, response.Id)
		require.Equal(t, "new_actor", response.Name)
		require.EqualValues(t, "portal", response.Role)
		require.NotZero(t, response.CreatedAt)

		// Verify the actor was actually created in the database
		createdActor, err := querier.ActorFindById(ctx, ds, response.Id)
		require.NoError(t, err)
		require.NotNil(t, createdActor)
		require.Equal(t, "new_actor", createdActor.Name)
		require.EqualValues(t, "portal", createdActor.Role)

		// Verify the associated queue was created
		queue, err := querier.QueueFindById(ctx, ds, createdActor.QueueID)
		require.NoError(t, err)
		require.NotNil(t, queue)
		require.Equal(t, "new_actor", queue.Name)
		require.Equal(t, []byte(`{"type": "actor"}`), queue.Metadata)
	})

	t.Run("Invalid body", func(t *testing.T) {
		resp, resBody := PostHttp(t, server.URL+"/v1/admin/actors", `{"invalid_json"}`, "admin-token")
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)

		resJson := testhelper.JsonToMap(t, resBody)
		require.Contains(t, resJson, "error")
		require.Contains(t, resJson["error"], "invalid character")
	})

	t.Run("Missing required fields", func(t *testing.T) {
		resp, resBody := PostHttp(t, server.URL+"/v1/admin/actors", `{}`, "admin-token")
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)

		resJson := testhelper.JsonToMap(t, resBody)
		require.Contains(t, resJson, "error")
		require.Contains(t, resJson["error"], "Missing required field: name")
	})

	t.Run("Non-admin token", func(t *testing.T) {
		resp, _ := PostHttp(t, server.URL+"/v1/admin/actors", `{"name":"new_actor"}`, "actor-token")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Invalid token", func(t *testing.T) {
		resp, _ := PostHttp(t, server.URL+"/v1/admin/actors", `{"name":"new_actor"}`, "invalid_token")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestAdminGetActorEndpoint(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	server, ds, _ := SetupHttpTestWithDb(t, ctx)

	actor1 := fixture.InsertActor(t, ctx, ds, "actor1")
	actor2 := fixture.InsertActor(t, ctx, ds, "actor2")
	fixture.InsertToken(t, ctx, ds, "admin-token", actor1.ID, 0, []string{"admin"})
	fixture.InsertToken(t, ctx, ds, "actor-token", actor2.ID, 0, []string{"user"})

	t.Run("Valid admin token and existing actor", func(t *testing.T) {
		resp, resBody := GetHttp(t, fmt.Sprintf("%s/v1/admin/actors/%d", server.URL, actor1.ID), "admin-token")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var response api.AdminGetActor200JSONResponse
		err := json.Unmarshal([]byte(resBody), &response)
		require.NoError(t, err)

		require.Equal(t, actor1.ID, response.Data.Id)
		require.Equal(t, "actor1", response.Data.Name)
		require.NotZero(t, response.Data.CreatedAt)
	})

	t.Run("Valid admin token but non-existent actor", func(t *testing.T) {
		resp, _ := GetHttp(t, fmt.Sprintf("%s/v1/admin/actors/%d", server.URL, 999), "admin-token")
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Non-admin token", func(t *testing.T) {
		resp, _ := GetHttp(t, fmt.Sprintf("%s/v1/admin/actors/%d", server.URL, actor1.ID), "actor-token")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Invalid token", func(t *testing.T) {
		resp, _ := GetHttp(t, fmt.Sprintf("%s/v1/admin/actors/%d", server.URL, actor1.ID), "invalid_token")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestAdminUpdateActorEndpoint(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	server, ds, _ := SetupHttpTestWithDb(t, ctx)

	actor1 := fixture.InsertActor(t, ctx, ds, "actor1")
	actor2 := fixture.InsertActor(t, ctx, ds, "actor2")
	fixture.InsertToken(t, ctx, ds, "admin-token", actor1.ID, 0, []string{"admin"})
	fixture.InsertToken(t, ctx, ds, "actor-token", actor2.ID, 0, []string{"user"})

	t.Run("Valid admin token and existing actor", func(t *testing.T) {
		resp, resBody := PatchHttp(t, fmt.Sprintf("%s/v1/admin/actors/%d", server.URL, actor1.ID), `{"name":"updated_actor","role":"portal"}`, "admin-token")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var response api.AdminUpdateActor200JSONResponse
		err := json.Unmarshal([]byte(resBody), &response)
		require.NoError(t, err)

		require.Equal(t, actor1.ID, response.Data.Id)
		require.Equal(t, "updated_actor", response.Data.Name)
		require.EqualValues(t, "portal", response.Data.Role)
		require.NotZero(t, response.Data.CreatedAt)

		// Verify the actor was actually updated in the database
		updatedActor, err := querier.ActorFindById(ctx, ds, actor1.ID)
		require.NoError(t, err)
		require.NotNil(t, updatedActor)
		require.Equal(t, "updated_actor", updatedActor.Name)
		require.EqualValues(t, "portal", updatedActor.Role)
	})

	t.Run("Valid admin token but non-existent actor", func(t *testing.T) {
		resp, _ := PatchHttp(t, fmt.Sprintf("%s/v1/admin/actors/%d", server.URL, 999), `{"name":"updated_actor"}`, "admin-token")
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Invalid body", func(t *testing.T) {
		resp, resBody := PatchHttp(t, fmt.Sprintf("%s/v1/admin/actors/%d", server.URL, actor1.ID), `{"invalid_json"}`, "admin-token")
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)

		resJson := testhelper.JsonToMap(t, resBody)
		require.Contains(t, resJson, "error")
		require.Contains(t, resJson["error"], "invalid character")
	})

	t.Run("Non-admin token", func(t *testing.T) {
		resp, _ := PatchHttp(t, fmt.Sprintf("%s/v1/admin/actors/%d", server.URL, actor1.ID), `{"name":"updated_actor"}`, "actor-token")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Invalid token", func(t *testing.T) {
		resp, _ := PatchHttp(t, fmt.Sprintf("%s/v1/admin/actors/%d", server.URL, actor1.ID), `{"name":"updated_actor"}`, "invalid_token")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestAdminDeleteActorEndpoint(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	server, ds, _ := SetupHttpTestWithDb(t, ctx)

	actor1 := fixture.InsertActor(t, ctx, ds, "actor1")
	adminActor := fixture.InsertActor(t, ctx, ds, "actor2")
	fixture.InsertToken(t, ctx, ds, "admin-token", adminActor.ID, 0, []string{"admin"})

	t.Run("Valid admin token and existing actor", func(t *testing.T) {
		resp, _ := DeleteHttp(t, fmt.Sprintf("%s/v1/admin/actors/%d", server.URL, actor1.ID), "admin-token")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify the actor was actually deleted from the database
		_, err := querier.ActorFindById(ctx, ds, actor1.ID)
		require.Error(t, err)
		require.IsType(t, sql.ErrNoRows, err)
	})

	t.Run("Valid admin token but non-existent actor", func(t *testing.T) {
		resp, _ := DeleteHttp(t, fmt.Sprintf("%s/v1/admin/actors/%d", server.URL, 999), "admin-token")
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Non-admin token", func(t *testing.T) {
		resp, _ := DeleteHttp(t, fmt.Sprintf("%s/v1/admin/actors/%d", server.URL, actor1.ID), "actor-token")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Invalid token", func(t *testing.T) {
		resp, _ := DeleteHttp(t, fmt.Sprintf("%s/v1/admin/actors/%d", server.URL, actor1.ID), "invalid_token")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Actor with associated configs", func(t *testing.T) {
		actorWithConfig := fixture.InsertActor(t, ctx, ds, "actor_with_config")
		fixture.InsertConfig(t, ctx, ds, actorWithConfig.ID, map[string]string{"key": "value"})

		resp, _ := DeleteHttp(t, fmt.Sprintf("%s/v1/admin/actors/%d", server.URL, actorWithConfig.ID), "admin-token")
		require.Equal(t, http.StatusConflict, resp.StatusCode)
	})
}
