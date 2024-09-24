package apitest

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
	"gitlab.com/navyx/ai/maos/maos-core/internal/fixture"
	"gitlab.com/navyx/ai/maos/maos-core/internal/testhelper"
)

func TestInvocationCreateEndpoint(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name           string
		body           string
		agentName      string
		tokenName      string
		permissions    []string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Successful invocation insertion",
			body:           `{"agent":"agent1","meta":{"kind": "test"},"payload":{"key1": 16888,"key2":{"key3": "value3"}}}`,
			agentName:      "agent1",
			tokenName:      "token001",
			permissions:    []string{"create:invocation"},
			expectedStatus: http.StatusCreated,
			expectedBody:   `{"id":"(ignore)"}`,
		},
		{
			name:           "No payload in request",
			body:           `{"agent":"agent1","meta":{"kind": "test"}}`,
			agentName:      "agent1",
			tokenName:      "token007",
			permissions:    []string{"create:invocation"},
			expectedStatus: http.StatusCreated,
			expectedBody:   `{"id":"(ignore)"}`,
		},
		{
			name:           "Missing agent in request",
			body:           `{"meta":{"kind": "test"},"payload":{"key1": 16888,"key2":{"key3": "value3"}}}`,
			agentName:      "agent1",
			tokenName:      "token001",
			permissions:    []string{"create:invocation"},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "agent not found",
		},
		{
			name:           "No permission to create invocation",
			body:           `{"agent":"agent1","meta":{"kind": "test"},"payload":{"key1": 16888,"key2":{"key3": "value3"}}}`,
			agentName:      "agent1",
			tokenName:      "token002",
			permissions:    []string{"read:invocation"},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "",
		},
		{
			name:           "Invalid JSON in request body",
			body:           `{"agent":"agent1","meta":{"kind": "test"},"payload":{"key1": 16888,"key2":{"key3": "value3"}}`,
			agentName:      "agent1",
			tokenName:      "token003",
			permissions:    []string{"create:invocation"},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "can't decode JSON",
		},
		{
			name:           "Empty request body",
			body:           ``,
			agentName:      "agent1",
			tokenName:      "token004",
			permissions:    []string{"create:invocation"},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "can't decode JSON",
		},
		{
			name:           "Agent not found",
			body:           `{"agent":"non_existent_agent","meta":{"kind": "test"},"payload":{"key1": 16888,"key2":{"key3": "value3"}}}`,
			agentName:      "agent1",
			tokenName:      "token005",
			permissions:    []string{"create:invocation"},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "agent not found",
		},
		{
			name:           "Missing metadata in request",
			body:           `{"agent":"agent1","payload":{"key1": 16888,"key2":{"key3": "value3"}}}`,
			agentName:      "agent1",
			tokenName:      "token006",
			permissions:    []string{"create:invocation"},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Meta is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, accessor, _ := SetupHttpTestWithDb(t, ctx)

			agent := fixture.InsertAgent(t, ctx, accessor.Source(), tt.agentName)
			token := fixture.InsertToken(t, ctx, accessor.Source(), tt.tokenName, agent.ID, 0, tt.permissions)

			resp, resBody := PostHttp(t, server.URL+"/v1/invocations/async", tt.body, token.ID)
			require.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusCreated {
				invocations, err := accessor.Querier().InvocationGetAvailable(ctx, accessor.Source(), &dbsqlc.InvocationGetAvailableParams{
					AttemptedBy: agent.ID,
					QueueID:     agent.QueueID,
					Max:         10,
				})
				require.NoError(t, err)
				require.Len(t, invocations, 1)
				require.JSONEq(t, fmt.Sprintf(`{"id":"%d"}`, invocations[0].ID), resBody)
				require.Equal(t, invocations[0].AttemptedBy, []int64{agent.ID})
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

func TestInvocationGetEndpoint(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	processInvocation := func(t *testing.T, ctx context.Context, server *httptest.Server, token string, respond string) {
		resp, resBody := GetHttp(t, server.URL+"/v1/invocations/next", token)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		bodyJson := testhelper.JsonToMap(t, resBody)
		switch respond {
		case "not_respond":
		case "completed":
			body := fmt.Sprintf(`{"result":{"res":"%s"}}`, bodyJson["payload"].(map[string]interface{})["req"].(string))
			resp, resBody = PostHttp(t, fmt.Sprintf("%s/v1/invocations/%s/response", server.URL, bodyJson["id"].(string)), body, token)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		case "error":
			body := fmt.Sprintf(`{"errors":{"err":"%s"}}`, bodyJson["payload"].(map[string]interface{})["req"].(string))
			resp, resBody = PostHttp(t, fmt.Sprintf("%s/v1/invocations/%s/error", server.URL, bodyJson["id"].(string)), body, token)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		}
	}

	setup := func(t *testing.T, ctx context.Context) (*httptest.Server, string, string, string, *httptest.Server) {
		server, accessor, server2 := SetupHttpTestWithDb(t, ctx)
		agent := fixture.InsertAgent(t, ctx, accessor.Source(), "agent1")
		token := fixture.InsertToken(t, ctx, accessor.Source(), "agent-token", agent.ID, 0, []string{"read:invocation"})
		user := fixture.InsertAgent(t, ctx, accessor.Source(), "user")
		userToken := fixture.InsertToken(t, ctx, accessor.Source(), "user-token", user.ID, 0, []string{"create:invocation"})

		body := `{"agent":"agent1","meta":{"kind": "test"},"payload":{"req": "16888"}}`
		resp, respBody := PostHttp(t, server.URL+"/v1/invocations/async", body, userToken.ID)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		id := testhelper.JsonToMap(t, respBody)["id"].(string)

		return server, token.ID, userToken.ID, id, server2
	}

	t.Run("invocation completed", func(t *testing.T) {
		server, token, userToken, invId, server2 := setup(t, ctx)

		processInvocation(t, ctx, server2, token, "completed")

		resp, respBody := GetHttp(t, server.URL+"/v1/invocations/"+invId, userToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		testhelper.AssertJsonEqIgnoringFields(t,
			fmt.Sprintf(`{"id":"%s","state":"completed","result":{"res":"16888"},"attempted_at":0,"finalized_at":0}`, invId),
			respBody,
			"attempted_at",
			"finalized_at",
		)
	})

	t.Run("get non-running invocation without waiting", func(t *testing.T) {
		server, _, userToken, invId, _ := setup(t, ctx)

		resp, respBody := GetHttp(t, server.URL+"/v1/invocations/"+invId, userToken)
		require.Equal(t, http.StatusAccepted, resp.StatusCode)
		testhelper.AssertJsonEqIgnoringFields(t,
			fmt.Sprintf(`{"id":"%s","state":"available"}`, invId),
			respBody,
			"attempted_at",
			"finalized_at",
		)
	})

	t.Run("get running invocation without waiting", func(t *testing.T) {
		server, token, userToken, invId, server2 := setup(t, ctx)

		processInvocation(t, ctx, server2, token, "not_respond")

		resp, respBody := GetHttp(t, server.URL+"/v1/invocations/"+invId, userToken)
		require.Equal(t, http.StatusAccepted, resp.StatusCode)
		testhelper.AssertJsonEqIgnoringFields(t,
			fmt.Sprintf(`{"id":"%s","state":"running","attempted_at":0}`, invId),
			respBody,
			"attempted_at",
			"finalized_at",
		)
	})

	t.Run("get running invocation with waiting", func(t *testing.T) {
		server, token, userToken, invId, server2 := setup(t, ctx)

		processInvocation(t, ctx, server2, token, "not_respond")

		start := time.Now()
		resp, respBody := GetHttp(t, server.URL+"/v1/invocations/"+invId+"?wait=1", userToken)
		require.Equal(t, http.StatusAccepted, resp.StatusCode)
		testhelper.AssertJsonEqIgnoringFields(t,
			fmt.Sprintf(`{"id":"%s","state":"running","attempted_at":0}`, invId),
			respBody,
			"attempted_at",
			"finalized_at",
		)
		require.InDelta(t, 1000, time.Since(start).Milliseconds(), 100)
	})

	t.Run("get completed invocation with waiting", func(t *testing.T) {
		server, token, userToken, invId, server2 := setup(t, ctx)

		go func() {
			time.Sleep(500 * time.Millisecond)
			processInvocation(t, ctx, server2, token, "completed")
		}()

		start := time.Now()
		resp, respBody := GetHttp(t, server.URL+"/v1/invocations/"+invId+"?wait=1", userToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		testhelper.AssertJsonEqIgnoringFields(t,
			fmt.Sprintf(`{"id":"%s","state":"completed","result":{"res":"16888"},"attempted_at":0,"finalized_at":0}`, invId),
			respBody,
			"attempted_at",
			"finalized_at",
		)
		require.InDelta(t, 500, time.Since(start).Milliseconds(), 100)
	})

	t.Run("get error invocation with waiting", func(t *testing.T) {
		server, token, userToken, invId, server2 := setup(t, ctx)

		go func() {
			time.Sleep(500 * time.Millisecond)
			processInvocation(t, ctx, server2, token, "error")
		}()

		start := time.Now()
		resp, respBody := GetHttp(t, server.URL+"/v1/invocations/"+invId+"?wait=1", userToken)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		testhelper.AssertJsonEqIgnoringFields(t,
			fmt.Sprintf(`{"id":"%s","state":"discarded","errors":{"err":"16888"},"attempted_at":0,"finalized_at":0}`, invId),
			respBody,
			"attempted_at",
			"finalized_at",
		)
		require.InDelta(t, 500, time.Since(start).Milliseconds(), 200)
	})
}
