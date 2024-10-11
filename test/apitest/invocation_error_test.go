package apitest

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
	"gitlab.com/navyx/ai/maos/maos-core/internal/fixture"
)

func TestInvocationReturnErrorEndpoint(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	setup := func(t *testing.T, ctx context.Context) (*httptest.Server, dbaccess.DataSource, *dbsqlc.Actor, *dbsqlc.ApiToken) {
		server, ds, _ := SetupHttpTestWithDb(t, ctx)
		actor := fixture.InsertActor(t, ctx, ds, "test-actor")
		token := fixture.InsertToken(t, ctx, ds, "actor-token", actor.ID, 0, []string{"read:invocation"})
		return server, ds, actor, token
	}

	t.Run("running invocations exist", func(t *testing.T) {
		server, ds, actor, token := setup(t, ctx)

		// insert and change state to running
		invocation := fixture.InsertInvocation(t, ctx, ds, "available", `{"seq": 1}`, actor.Name)
		_, err := querier.InvocationGetAvailable(ctx, ds, &dbsqlc.InvocationGetAvailableParams{
			AttemptedBy: actor.ID,
			QueueID:     actor.QueueID,
			Max:         1,
		})
		require.NoError(t, err)

		body := `{"errors":{"err": 16888}}`
		resp, _ := PostHttp(t, fmt.Sprintf("%s/v1/invocations/%d/error", server.URL, invocation), body, token.ID)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		row, err := querier.InvocationFindById(ctx, ds, invocation)
		require.NoError(t, err)
		require.Equal(t, dbsqlc.InvocationState("discarded"), row.State)
		require.JSONEq(t, `{"err": 16888}`, string(row.Errors))
	})

	t.Run("invalid token", func(t *testing.T) {
		server, _, _, token := setup(t, ctx)

		body := `{"errors":{"err": 16888}}`
		resp, _ := PostHttp(t, fmt.Sprintf("%s/v1/invocations/%d/error", server.URL, 1998), body, token.ID+"n")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("invalid permission", func(t *testing.T) {
		server, ds, actor, _ := setup(t, ctx)
		token := fixture.InsertToken(t, ctx, ds, "actor-token2", actor.ID, 0, []string{"create:invocation"})

		body := `{"errors":{"err": 16888}}`
		resp, _ := PostHttp(t, fmt.Sprintf("%s/v1/invocations/%d/error", server.URL, 1998), body, token.ID+"n")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("invalid invocation id", func(t *testing.T) {
		server, _, _, token := setup(t, ctx)

		body := `{"errors":{"err": 16888}}`
		resp, _ := PostHttp(t, fmt.Sprintf("%s/v1/invocations/%d/error", server.URL, 1998), body, token.ID)
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("attempted_by mismatch", func(t *testing.T) {
		server, ds, actor, _ := setup(t, ctx)
		actor2 := fixture.InsertActor(t, ctx, ds, "test-actor2")
		token2 := fixture.InsertToken(t, ctx, ds, "actor2-token", actor2.ID, 0, []string{"read:invocation"})

		// insert and change state to running
		invocation := fixture.InsertInvocation(t, ctx, ds, "available", `{"seq": 1}`, actor.Name)
		_, err := querier.InvocationGetAvailable(ctx, ds, &dbsqlc.InvocationGetAvailableParams{
			AttemptedBy: actor.ID,
			QueueID:     actor.QueueID,
			Max:         1,
		})
		require.NoError(t, err)

		body := `{"errors":{"err": 16888}}`
		resp, _ := PostHttp(t, fmt.Sprintf("%s/v1/invocations/%d/error", server.URL, invocation), body, token2.ID)
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}
