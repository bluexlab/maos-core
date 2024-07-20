package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"gitlab.com/navyx/ai/maos/maos-core/api"
)

type TestAPIHandler struct {
	APIHandler

	NewContext context.Context
}

// GetCallerConfig implements the GET /v1/config endpoint
func (s *TestAPIHandler) GetCallerConfig(ctx context.Context, request api.GetCallerConfigRequestObject) (api.GetCallerConfigResponseObject, error) {
	s.NewContext = ctx
	config := api.GetCallerConfig200JSONResponse{
		"config1": "value1",
		"config2": "value2",
	}
	return config, nil
}

func TestBearerAuth(t *testing.T) {
	t.Parallel()

	server := &TestAPIHandler{}
	middlewares := []api.StrictMiddlewareFunc{NewBearerAuthMiddleware(
		func(ctx context.Context, token string) (context.Context, error) {
			if token == "valid_token" {
				newContext := context.WithValue(ctx, "token", map[string]interface{}{"permissions": []string{"read"}})
				return newContext, nil
			}
			return ctx, fmt.Errorf("invalid token")
		})}

	handler := api.NewStrictHandler(server, middlewares)

	// Create a new router and register the handlers
	r := mux.NewRouter()
	api.HandlerWithOptions(handler, api.GorillaServerOptions{
		BaseRouter: r,
	})

	// Create a test server
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Test cases
	testCases := []struct {
		name           string
		token          string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid token",
			token:          "valid_token",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"config1":"value1","config2":"value2"}`,
		},
		{
			name:           "Invalid token",
			token:          "invalid_token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"invalid token"}`,
		},
		{
			name:           "Missing token",
			token:          "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"missing authorization header"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", ts.URL+"/v1/config", nil)
			if err != nil {
				t.Fatalf("Error creating request: %v", err)
			}

			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Error making request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, resp.StatusCode)
			}

			var body map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("Error decoding response body: %v", err)
			}

			expectedBody := make(map[string]interface{})
			if err := json.Unmarshal([]byte(tc.expectedBody), &expectedBody); err != nil {
				t.Fatalf("Error parsing expected body: %v", err)
			}

			if !compareJSON(body, expectedBody) {
				t.Errorf("Expected body %v, got %v", expectedBody, body)
			}
		})
	}
}

func TestBearerAuthWithValidator(t *testing.T) {
	t.Parallel()

	server := &TestAPIHandler{}
	middlewares := []api.StrictMiddlewareFunc{NewBearerAuthMiddleware(
		func(ctx context.Context, token string) (context.Context, error) {
			if token == "valid_token" {
				newContext := context.WithValue(ctx, "token", map[string]interface{}{"permissions": []string{"read"}})
				return newContext, nil
			}
			return ctx, fmt.Errorf("invalid token")
		})}

	handler := api.NewStrictHandler(server, middlewares)
	r := mux.NewRouter()
	api.HandlerWithOptions(handler, api.GorillaServerOptions{
		BaseRouter: r,
	})

	ts := httptest.NewServer(r)
	defer ts.Close()

	req, err := http.NewRequest("GET", ts.URL+"/v1/config", nil)
	if err != nil {
		t.Fatalf("Error creating request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer valid_token")
	resp, err := http.DefaultClient.Do(req)
	require.Nil(t, err)

	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t,
		server.NewContext.Value("token"),
		map[string]interface{}{"permissions": []string{"read"}},
		"Expected token to be stored in the context")
}

func compareJSON(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || v != w {
			return false
		}
	}
	return true
}

// TestSwaggerEndpoints remains the same as in the previous example
