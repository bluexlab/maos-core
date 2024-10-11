package apitest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
	"gitlab.com/navyx/ai/maos/maos-core/internal/fixture"
)

func TestInvocationGetNextEndpoint(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	setup := func(t *testing.T, ctx context.Context) (*httptest.Server, dbaccess.DataSource, *dbsqlc.Actor, *dbsqlc.ApiToken) {
		server, ds, _ := SetupHttpTestWithDb(t, ctx)
		actor := fixture.InsertActor(t, ctx, ds, "test-actor")
		token := fixture.InsertToken(t, ctx, ds, "actor-token", actor.ID, 0, []string{"read:invocation"})
		return server, ds, actor, token
	}

	t.Run("Invocations exist", func(t *testing.T) {
		server, ds, actor, token := setup(t, ctx)
		invocation := fixture.InsertInvocation(t, ctx, ds, "available", `{"seq": 1}`, actor.Name)

		resp, resBody := GetHttp(t, server.URL+"/v1/invocations/next", token.ID)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.JSONEq(t,
			fmt.Sprintf(`{"id":"%d", "meta":{"kind":"test","trace_id":"123"}, "payload":{"seq":1}}`, invocation),
			resBody)

		row, err := querier.InvocationFindById(ctx, ds, invocation)
		require.NoError(t, err)
		require.Equal(t, dbsqlc.InvocationState("running"), row.State)
	})

	t.Run("Timeout", func(t *testing.T) {
		server, ds, actor, token := setup(t, ctx)
		fixture.InsertInvocation(t, ctx, ds, "running", `{"seq": 1}`, actor.Name)

		resp, _ := GetHttp(t, server.URL+"/v1/invocations/next?wait=1", token.ID)
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Available and running Invocations both exist", func(t *testing.T) {
		server, ds, actor, token := setup(t, ctx)
		fixture.InsertInvocation(t, ctx, ds, "running", `{"seq": 2}`, actor.Name)
		invocation := fixture.InsertInvocation(t, ctx, ds, "available", `{"seq": 1}`, actor.Name)
		fixture.InsertInvocation(t, ctx, ds, "running", `{"seq": 3}`, actor.Name)

		resp, resBody := GetHttp(t, server.URL+"/v1/invocations/next", token.ID)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.JSONEq(t,
			fmt.Sprintf(`{"id":"%d", "meta":{"kind":"test","trace_id":"123"}, "payload":{"seq":1}}`, invocation),
			resBody)
	})

	t.Run("Invocations insert after", func(t *testing.T) {
		server, ds, _ := SetupHttpTestWithDb(t, ctx)

		actor := fixture.InsertActor(t, ctx, ds, "test-actor")
		user := fixture.InsertActor(t, ctx, ds, "test-user")
		token := fixture.InsertToken(t, ctx, ds, "actor-token", actor.ID, 0, []string{"read:invocation", "create:invocation"})
		userToken := fixture.InsertToken(t, ctx, ds, "user-token", user.ID, 0, []string{"create:invocation"})

		var invocationId string
		go func() {
			time.Sleep(10 * time.Millisecond)

			body := `{"actor":"test-actor","meta":{"kind": "test", "trace_id": "456"},"payload":{"seq": 16888}}`
			resp, resBody := PostHttp(t, server.URL+"/v1/invocations/async", body, userToken.ID)
			require.Equal(t, http.StatusCreated, resp.StatusCode)
			var res map[string]interface{}
			require.NoError(t, json.Unmarshal([]byte(resBody), &res))
			invocationId = res["id"].(string)
		}()

		resp, resBody := GetHttp(t, server.URL+"/v1/invocations/next", token.ID)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.JSONEq(t,
			fmt.Sprintf(`{"id":"%s", "meta":{"kind":"test","trace_id":"456"}, "payload":{"seq":16888}}`, invocationId),
			resBody)
	})

	t.Run("multiple get next request", func(t *testing.T) {
		server, ds, _ := SetupHttpTestWithDb(t, ctx)

		actor := fixture.InsertActor(t, ctx, ds, "test-actor")
		user := fixture.InsertActor(t, ctx, ds, "test-user")
		token := fixture.InsertToken(t, ctx, ds, "actor-token", actor.ID, 0, []string{"read:invocation", "create:invocation"})
		userToken := fixture.InsertToken(t, ctx, ds, "user-token", user.ID, 0, []string{"create:invocation"})

		count := 10
		wg := sync.WaitGroup{}
		wg.Add(count)
		insert := func(i int) {
			body := fmt.Sprintf(`{"actor":"test-actor","meta":{"kind": "test", "trace_id": "456"},"payload":{"seq": "%d"}}`, i)
			resp, _ := PostHttp(t, server.URL+"/v1/invocations/async", body, userToken.ID)
			require.Equal(t, http.StatusCreated, resp.StatusCode)
			wg.Done()
		}

		resCh := make(chan string, 10)
		next := func() {
			resp, resBody := GetHttp(t, server.URL+"/v1/invocations/next", token.ID)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var res map[string]interface{}
			require.NoError(t, json.Unmarshal([]byte(resBody), &res))
			resCh <- res["payload"].(map[string]interface{})["seq"].(string)
		}

		for i := 0; i < count; i++ {
			go next()
		}

		for i := 0; i < count; i++ {
			go insert(i)
		}

		var invocationIds []string
		for i := 0; i < count; i++ {
			invocationIds = append(invocationIds, <-resCh)
		}

		wg.Wait()

		expected := lo.RepeatBy(count, func(i int) string { return fmt.Sprintf("%d", i) })
		require.ElementsMatch(t, expected, invocationIds)
	})
}
