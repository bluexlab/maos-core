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

func TestInvocationReturnResponseEndpoint(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	setup := func(t *testing.T, ctx context.Context) (*httptest.Server, dbaccess.Accessor, *dbsqlc.Agent, *dbsqlc.ApiToken) {
		server, accessor, _ := SetupHttpTestWithDb(t, ctx)
		agent := fixture.InsertAgent(t, ctx, accessor.Source(), "test-agent")
		token := fixture.InsertToken(t, ctx, accessor.Source(), "agent-token", agent.ID, 0, []string{"read:invocation"})
		return server, accessor, agent, token
	}

	t.Run("running invocations exist", func(t *testing.T) {
		server, accessor, agent, token := setup(t, ctx)

		// insert and change state to running
		invocation := fixture.InsertInvocation(t, ctx, accessor.Source(), "available", `{"seq": 1}`, agent.Name)
		_, err := accessor.Querier().InvocationGetAvailable(ctx, accessor.Source(), &dbsqlc.InvocationGetAvailableParams{
			AttemptedBy: agent.ID,
			QueueID:     agent.QueueID,
			Max:         1,
		})
		require.NoError(t, err)

		body := `{"result":{"res": 16888}}`
		resp, _ := PostHttp(t, fmt.Sprintf("%s/v1/invocations/%d/response", server.URL, invocation), body, token.ID)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		row, err := accessor.Querier().InvocationFindById(ctx, accessor.Source(), invocation)
		require.NoError(t, err)
		require.Equal(t, dbsqlc.InvocationState("completed"), row.State)
		require.JSONEq(t, `{"res": 16888}`, string(row.Result))
	})

	t.Run("invalid token", func(t *testing.T) {
		server, _, _, token := setup(t, ctx)

		body := `{"result":{"res": 16888}}`
		resp, _ := PostHttp(t, fmt.Sprintf("%s/v1/invocations/%d/response", server.URL, 1998), body, token.ID+"n")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("invalid permission", func(t *testing.T) {
		server, _, _, token := setup(t, ctx)

		body := `{"result":{"res": 16888}}`
		resp, _ := PostHttp(t, fmt.Sprintf("%s/v1/invocations/%d/response", server.URL, 1998), body, token.ID+"n")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("invalid invocation id", func(t *testing.T) {
		server, _, _, token := setup(t, ctx)

		body := `{"result":{"res": 16888}}`
		resp, _ := PostHttp(t, fmt.Sprintf("%s/v1/invocations/%d/response", server.URL, 1998), body, token.ID)
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("attempted_by mismatch", func(t *testing.T) {
		server, accessor, agent, _ := setup(t, ctx)
		agent2 := fixture.InsertAgent(t, ctx, accessor.Source(), "test-agent2")
		token2 := fixture.InsertToken(t, ctx, accessor.Source(), "agent2-token", agent2.ID, 0, []string{"read:invocation"})

		// insert and change state to running
		invocation := fixture.InsertInvocation(t, ctx, accessor.Source(), "available", `{"seq": 1}`, agent.Name)
		_, err := accessor.Querier().InvocationGetAvailable(ctx, accessor.Source(), &dbsqlc.InvocationGetAvailableParams{
			AttemptedBy: agent.ID,
			QueueID:     agent.QueueID,
			Max:         1,
		})
		require.NoError(t, err)

		body := `{"result":{"res": 16888}}`
		resp, _ := PostHttp(t, fmt.Sprintf("%s/v1/invocations/%d/response", server.URL, invocation), body, token2.ID)
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}
