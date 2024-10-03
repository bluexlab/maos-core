package apitest

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
	"gitlab.com/navyx/ai/maos/maos-core/internal/fixture"
	"gitlab.com/navyx/ai/maos/maos-core/internal/testhelper"
	"golang.org/x/exp/rand"
)

func TestInvocationExecuteEndpoint(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("one actor and one execution", func(t *testing.T) {
		server, accessor, _ := SetupHttpTestWithDb(t, ctx)

		actor := fixture.InsertActor(t, ctx, accessor.Source(), "actor1")
		token := fixture.InsertToken(t, ctx, accessor.Source(), "actor-token", actor.ID, 0, []string{"read:invocation"})
		user := fixture.InsertActor(t, ctx, accessor.Source(), "user")
		userToken := fixture.InsertToken(t, ctx, accessor.Source(), "user-token", user.ID, 0, []string{"create:invocation"})

		var invocationId int64
		go func() {
			// get available invocation
			time.Sleep(50 * time.Millisecond)
			invocations, err := accessor.Querier().InvocationGetAvailable(ctx, accessor.Source(), &dbsqlc.InvocationGetAvailableParams{
				AttemptedBy: actor.ID,
				QueueID:     actor.QueueID,
				Max:         1,
			})
			require.NoError(t, err)
			require.Len(t, invocations, 1)
			invocationId = invocations[0].ID
			require.JSONEq(t, `{"kind":"test","trace_id":"456"}`, string(invocations[0].Metadata))

			// set response for the invocation and set it to completed
			body := `{"result":{"seq": "16888"}}`
			resp, _ := PostHttp(t, fmt.Sprintf("%s/v1/invocations/%d/response", server.URL, invocationId), body, token.ID)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		}()

		body := `{"actor":"actor1","meta":{"kind": "test", "trace_id": "456"},"payload":{"key1": 16888,"key2":{"key3": "value3"}}}`
		resp, resBody := PostHttp(t, server.URL+"/v1/invocations/sync", body, userToken.ID)
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		resJson := testhelper.JsonToMap(t, resBody)
		require.Equal(t, strconv.FormatInt(invocationId, 10), resJson["id"])
		require.Equal(t, map[string]interface{}{"seq": "16888"}, resJson["result"])
		require.Equal(t, "completed", resJson["state"])
		require.Equal(t, map[string]interface{}{"kind": "test", "trace_id": "456"}, resJson["meta"].(map[string]interface{}))
	})

	t.Run("timeout", func(t *testing.T) {
		server, accessor, _ := SetupHttpTestWithDb(t, ctx)

		fixture.InsertActor(t, ctx, accessor.Source(), "actor1")
		user := fixture.InsertActor(t, ctx, accessor.Source(), "user")
		userToken := fixture.InsertToken(t, ctx, accessor.Source(), "user-token", user.ID, 0, []string{"create:invocation"})

		body := `{"actor":"actor1","meta":{"kind": "test", "trace_id": "456"},"payload":{"key1": 16888,"key2":{"key3": "value3"}}}`
		resp, body := PostHttp(t, server.URL+"/v1/invocations/sync?wait=1", body, userToken.ID)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		bodyJson := testhelper.JsonToMap(t, body)
		require.NotEmpty(t, bodyJson["id"])
		testhelper.AssertJsonEqIgnoringFields(t, `{"id":"`+bodyJson["id"].(string)+`","state":"available","meta":{"kind":"test","trace_id":"456"}}`, body, "result")
	})

	t.Run("multiple actors and executions", func(t *testing.T) {
		server, accessor, _ := SetupHttpTestWithDb(t, ctx)

		const (
			actorCount   = 4
			executeCount = 20
		)

		actor := fixture.InsertActor(t, ctx, accessor.Source(), "actor1")
		token := fixture.InsertToken(t, ctx, accessor.Source(), "actor-token", actor.ID, 0, []string{"read:invocation"})
		user := fixture.InsertActor(t, ctx, accessor.Source(), "user")
		userToken := fixture.InsertToken(t, ctx, accessor.Source(), "user-token", user.ID, 0, []string{"create:invocation"})

		errCh := make(chan error, executeCount)

		var remainingRequests atomic.Int64
		remainingRequests.Store(int64(executeCount))

		for i := 0; i < actorCount; i++ {
			go func() {
				for {
					// get available invocation
					resp, resBody := GetHttp(t, server.URL+"/v1/invocations/next", token.ID)
					if resp.StatusCode != http.StatusOK {
						t.Log("Wrong status code", resp.StatusCode)
						errCh <- fmt.Errorf("response status code is %d", resp.StatusCode)
						return
					}

					require.Equal(t, http.StatusOK, resp.StatusCode)
					bodyJson := testhelper.JsonToMap(t, resBody)

					// set response for the invocation and set it to completed
					body := fmt.Sprintf(`{"result":{"res":"%s"}}`, bodyJson["payload"].(map[string]interface{})["req"].(string))
					resp, resBody = PostHttp(t, fmt.Sprintf("%s/v1/invocations/%s/response", server.URL, bodyJson["id"].(string)), body, token.ID)
					if http.StatusOK != resp.StatusCode {
						errCh <- fmt.Errorf("response status code is %d", resp.StatusCode)
						return
					}

					if remainingRequests.Add(-1) <= 0 {
						return
					}
				}
			}()
		}

		resCh := make(chan string, executeCount)

		execute := func(i int) {
			// random sleep to simulate different execution times
			time.Sleep(time.Duration(rand.Intn(6)+5) * time.Millisecond)

			body := fmt.Sprintf(`{"actor":"actor1","meta":{"kind": "test", "trace_id": "789"},"payload":{"req": "%d"}}`, i)
			resp, resBody := PostHttp(t, server.URL+"/v1/invocations/sync", body, userToken.ID)
			if http.StatusCreated != resp.StatusCode {
				errCh <- fmt.Errorf("response status code is %d not 201", resp.StatusCode)
				return
			}

			resJson := testhelper.JsonToMap(t, resBody)
			if resJson["state"] != "completed" {
				errCh <- fmt.Errorf("state is not completed")
				return
			}

			resCh <- resJson["result"].(map[string]interface{})["res"].(string)
		}
		for i := 0; i < executeCount; i++ {
			go execute(i)
		}

		var results []string
		for i := 0; i < executeCount; i++ {
			select {
			case err := <-errCh:
				require.NoError(t, err)
			case res := <-resCh:
				results = append(results, res)
			}
		}

		require.ElementsMatch(t, lo.RepeatBy(executeCount, func(i int) string { return strconv.Itoa(i) }), results)
	})
}

func TestInvocationExecuteEndpointErrorCases(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name           string
		body           string
		actorName      string
		tokenName      string
		permissions    []string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Missing actor in request",
			body:           `{"meta":{"kind": "test"},"payload":{"key1": 16888,"key2":{"key3": "value3"}}}`,
			actorName:      "actor1",
			tokenName:      "token001",
			permissions:    []string{"create:invocation"},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "actor not found",
		},
		{
			name:           "No permission to create invocation",
			body:           `{"actor":"actor1","meta":{"kind": "test"},"payload":{"key1": 16888,"key2":{"key3": "value3"}}}`,
			actorName:      "actor1",
			tokenName:      "token002",
			permissions:    []string{"read:invocation"},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "",
		},
		{
			name:           "Invalid JSON in request body",
			body:           `{"actor":"actor1","meta":{"kind": "test"},"payload":{"key1": 16888,"key2":{"key3": "value3"}}`,
			actorName:      "actor1",
			tokenName:      "token003",
			permissions:    []string{"create:invocation"},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "can't decode JSON",
		},
		{
			name:           "Empty request body",
			body:           ``,
			actorName:      "actor1",
			tokenName:      "token004",
			permissions:    []string{"create:invocation"},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "can't decode JSON",
		},
		{
			name:           "Actor not found",
			body:           `{"actor":"non_existent_actor","meta":{"kind": "test"},"payload":{"key1": 16888,"key2":{"key3": "value3"}}}`,
			actorName:      "actor1",
			tokenName:      "token005",
			permissions:    []string{"create:invocation"},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "actor not found",
		},
		{
			name:           "Missing metadata in request",
			body:           `{"actor":"actor1","payload":{"key1": 16888,"key2":{"key3": "value3"}}}`,
			actorName:      "actor1",
			tokenName:      "token006",
			permissions:    []string{"create:invocation"},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Meta is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, accessor, _ := SetupHttpTestWithDb(t, ctx)

			actor := fixture.InsertActor(t, ctx, accessor.Source(), tt.actorName)
			token := fixture.InsertToken(t, ctx, accessor.Source(), tt.tokenName, actor.ID, 0, tt.permissions)

			resp, resBody := PostHttp(t, server.URL+"/v1/invocations/sync", tt.body, token.ID)
			require.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusCreated {
				invocations, err := accessor.Querier().InvocationGetAvailable(ctx, accessor.Source(), &dbsqlc.InvocationGetAvailableParams{
					AttemptedBy: actor.ID,
					QueueID:     actor.QueueID,
					Max:         10,
				})
				require.NoError(t, err)
				require.Len(t, invocations, 1)
				require.JSONEq(t, fmt.Sprintf(`{"id":"%d"}`, invocations[0].ID), resBody)
				require.Equal(t, invocations[0].AttemptedBy, []int64{actor.ID})
			} else {
				if tt.expectedBody != "" {
					resJson := testhelper.JsonToMap(t, resBody)
					require.Contains(t, resJson, "error")
					require.Contains(t, resJson["error"], tt.expectedBody)
				}
			}
		})
	}
}
