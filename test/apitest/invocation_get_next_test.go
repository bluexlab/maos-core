package apitest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
	"gitlab.com/navyx/ai/maos/maos-core/internal/fixture"
)

func TestInvocationGetNextEndpoint(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("Invocations exist", func(t *testing.T) {
		server, accessor := SetupHttpTestWithDb(t, ctx)

		agent := fixture.InsertAgent(t, ctx, accessor.Source(), "test-agent")
		token := fixture.InsertToken(t, ctx, accessor.Source(), "agent-token", agent.ID, 0, []string{"read:invocation"})
		invocation := fixture.InsertInvocation(t, ctx, accessor.Source(), "available", `{"seq": 1}`, agent.Name)

		resp, resBody := GetHttp(t, server.URL+"/v1/invocations/next", token.ID)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.JSONEq(t,
			fmt.Sprintf(`{"id":"%d", "meta":{"kind":"test"}, "payload":{"seq":1}}`, invocation),
			resBody)

		row, err := accessor.Querier().InvocationFindById(ctx, accessor.Source(), invocation)
		require.NoError(t, err)
		require.Equal(t, dbsqlc.InvocationState("running"), row.State)
	})

	t.Run("Timeout", func(t *testing.T) {
		server, accessor := SetupHttpTestWithDb(t, ctx)

		agent := fixture.InsertAgent(t, ctx, accessor.Source(), "test-agent")
		token := fixture.InsertToken(t, ctx, accessor.Source(), "agent-token", agent.ID, 0, []string{"read:invocation"})
		fixture.InsertInvocation(t, ctx, accessor.Source(), "running", `{"seq": 1}`, agent.Name)

		resp, _ := GetHttp(t, server.URL+"/v1/invocations/next?wait=1", token.ID)
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Available and running Invocations both exist", func(t *testing.T) {
		server, accessor := SetupHttpTestWithDb(t, ctx)

		agent := fixture.InsertAgent(t, ctx, accessor.Source(), "test-agent")
		token := fixture.InsertToken(t, ctx, accessor.Source(), "agent-token", agent.ID, 0, []string{"read:invocation"})
		fixture.InsertInvocation(t, ctx, accessor.Source(), "running", `{"seq": 2}`, agent.Name)
		invocation := fixture.InsertInvocation(t, ctx, accessor.Source(), "available", `{"seq": 1}`, agent.Name)
		fixture.InsertInvocation(t, ctx, accessor.Source(), "running", `{"seq": 3}`, agent.Name)

		resp, resBody := GetHttp(t, server.URL+"/v1/invocations/next", token.ID)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.JSONEq(t,
			fmt.Sprintf(`{"id":"%d", "meta":{"kind":"test"}, "payload":{"seq":1}}`, invocation),
			resBody)
	})

	t.Run("Invocations insert after", func(t *testing.T) {
		server, accessor := SetupHttpTestWithDb(t, ctx)

		agent := fixture.InsertAgent(t, ctx, accessor.Source(), "test-agent")
		user := fixture.InsertAgent(t, ctx, accessor.Source(), "test-user")
		token := fixture.InsertToken(t, ctx, accessor.Source(), "agent-token", agent.ID, 0, []string{"read:invocation", "create:invocation"})
		userToken := fixture.InsertToken(t, ctx, accessor.Source(), "user-token", user.ID, 0, []string{"create:invocation"})

		var invocationId string
		go func() {
			time.Sleep(10 * time.Millisecond)

			body := `{"agent":"test-agent","meta":{"kind": "test"},"payload":{"seq": 16888}}`
			resp, resBody := PostHttp(t, server.URL+"/v1/invocations/async", body, userToken.ID)
			require.Equal(t, http.StatusCreated, resp.StatusCode)
			var res map[string]interface{}
			require.NoError(t, json.Unmarshal([]byte(resBody), &res))
			invocationId = res["id"].(string)
		}()

		resp, resBody := GetHttp(t, server.URL+"/v1/invocations/next", token.ID)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.JSONEq(t,
			fmt.Sprintf(`{"id":"%s", "meta":{"kind":"test"}, "payload":{"seq":16888}}`, invocationId),
			resBody)
	})

	t.Run("multiple get next request", func(t *testing.T) {
		server, accessor := SetupHttpTestWithDb(t, ctx)

		agent := fixture.InsertAgent(t, ctx, accessor.Source(), "test-agent")
		user := fixture.InsertAgent(t, ctx, accessor.Source(), "test-user")
		token := fixture.InsertToken(t, ctx, accessor.Source(), "agent-token", agent.ID, 0, []string{"read:invocation", "create:invocation"})
		userToken := fixture.InsertToken(t, ctx, accessor.Source(), "user-token", user.ID, 0, []string{"create:invocation"})

		count := 10
		wg := sync.WaitGroup{}
		wg.Add(count)
		insert := func(i int) {
			body := fmt.Sprintf(`{"agent":"test-agent","meta":{"kind": "test"},"payload":{"seq": "%d"}}`, i)
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
