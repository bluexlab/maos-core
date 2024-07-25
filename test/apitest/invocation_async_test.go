package apitest

import (
	"context"
	"fmt"
	"net/http"
	"testing"

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
			server, accessor := SetupHttpTestWithDb(t, ctx)

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
