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
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/internal/fixture"
	"gitlab.com/navyx/ai/maos/maos-core/internal/testhelper"
)

func TestAdminListActorEndpoint(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tests := []struct {
		name           string
		token          string
		queryParams    string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid admin token",
			token:          "admin-token",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"data":[{"id":1,"name":"actor1"},{"id":2,"name":"actor2"}],"meta":{"total_pages":1}}`,
		},
		{
			name:           "Valid admin token with pagination",
			token:          "admin-token",
			queryParams:    "?page=1&page_size=1",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"data":[{"id":1,"name":"actor1"}],"meta":{"total_pages":2}}`,
		},
		{
			name:           "Non-admin token",
			token:          "actor-token",
			queryParams:    "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "",
		},
		{
			name:           "Invalid token",
			token:          "invalid_token",
			queryParams:    "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, accessor, _ := SetupHttpTestWithDb(t, ctx)

			actor1 := fixture.InsertActor(t, ctx, accessor.Source(), "actor1")
			actor2 := fixture.InsertActor(t, ctx, accessor.Source(), "actor2")
			fixture.InsertToken(t, ctx, accessor.Source(), "admin-token", actor1.ID, 0, []string{"admin"})
			fixture.InsertToken(t, ctx, accessor.Source(), "actor-token", actor2.ID, 0, []string{"user"})

			resp, resBody := GetHttp(t, server.URL+"/v1/admin/actors"+tt.queryParams, tt.token)
			require.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusOK {
				var response api.AdminListActors200JSONResponse
				err := json.Unmarshal([]byte(resBody), &response)
				require.NoError(t, err)

				expectedResponse := api.AdminListActors200JSONResponse{}
				err = json.Unmarshal([]byte(tt.expectedBody), &expectedResponse)
				require.NoError(t, err)

				require.Equal(t, len(expectedResponse.Data), len(response.Data))
				require.Equal(t, expectedResponse.Meta.TotalPages, response.Meta.TotalPages)

				for i, expectedActor := range expectedResponse.Data {
					require.Equal(t, expectedActor.Name, response.Data[i].Name)
					require.NotZero(t, response.Data[i].Id)
					require.NotZero(t, response.Data[i].CreatedAt)
				}
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

func TestAdminCreateActorEndpoint(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tests := []struct {
		name           string
		body           string
		token          string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid admin token",
			body:           `{"name":"new_actor"}`,
			token:          "admin-token",
			expectedStatus: http.StatusCreated,
			expectedBody:   `{"id":3,"name":"new_actor"}`,
		},
		{
			name:           "Invalid body",
			body:           `{"invalid_json"}`,
			token:          "admin-token",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid character",
		},
		{
			name:           "Missing required fields",
			body:           `{}`,
			token:          "admin-token",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing required field: name",
		},
		{
			name:           "Non-admin token",
			body:           `{"name":"new_actor"}`,
			token:          "actor-token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "",
		},
		{
			name:           "Invalid token",
			body:           `{"name":"new_actor"}`,
			token:          "invalid_token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, accessor, _ := SetupHttpTestWithDb(t, ctx)

			actor1 := fixture.InsertActor(t, ctx, accessor.Source(), "actor1")
			actor2 := fixture.InsertActor(t, ctx, accessor.Source(), "actor2")
			fixture.InsertToken(t, ctx, accessor.Source(), "admin-token", actor1.ID, 0, []string{"admin"})
			fixture.InsertToken(t, ctx, accessor.Source(), "actor-token", actor2.ID, 0, []string{"user"})

			resp, resBody := PostHttp(t, server.URL+"/v1/admin/actors", tt.body, tt.token)
			require.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusCreated {
				var response api.AdminCreateActor201JSONResponse
				err := json.Unmarshal([]byte(resBody), &response)
				require.NoError(t, err)

				require.NotZero(t, response.Id)
				require.Equal(t, "new_actor", response.Name)
				require.NotZero(t, response.CreatedAt)

				// Verify the actor was actually created in the database
				createdActor, err := accessor.Querier().ActorFindById(ctx, accessor.Source(), response.Id)
				require.NoError(t, err)
				require.NotNil(t, createdActor)
				require.Equal(t, "new_actor", createdActor.Name)

				// Verify the associated queue was created
				queue, err := accessor.Querier().QueueFindById(ctx, accessor.Source(), createdActor.QueueID)
				require.NoError(t, err)
				require.NotNil(t, queue)
				require.Equal(t, "new_actor", queue.Name)
				require.Equal(t, []byte(`{"type": "actor"}`), queue.Metadata)
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

func TestAdminGetActorEndpoint(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tests := []struct {
		name           string
		actorID        int64
		token          string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid admin token and existing actor",
			actorID:        1,
			token:          "admin-token",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"data":{"id":1,"name":"actor1"}}`,
		},
		{
			name:           "Valid admin token but non-existent actor",
			actorID:        999,
			token:          "admin-token",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "",
		},
		{
			name:           "Non-admin token",
			actorID:        1,
			token:          "actor-token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "",
		},
		{
			name:           "Invalid token",
			actorID:        1,
			token:          "invalid_token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, accessor, _ := SetupHttpTestWithDb(t, ctx)

			actor1 := fixture.InsertActor(t, ctx, accessor.Source(), "actor1")
			actor2 := fixture.InsertActor(t, ctx, accessor.Source(), "actor2")
			fixture.InsertToken(t, ctx, accessor.Source(), "admin-token", actor1.ID, 0, []string{"admin"})
			fixture.InsertToken(t, ctx, accessor.Source(), "actor-token", actor2.ID, 0, []string{"user"})

			resp, resBody := GetHttp(t, fmt.Sprintf("%s/v1/admin/actors/%d", server.URL, tt.actorID), tt.token)
			require.Equal(t, tt.expectedStatus, resp.StatusCode)

			switch tt.expectedStatus {
			case http.StatusOK:
				var response api.AdminGetActor200JSONResponse
				err := json.Unmarshal([]byte(resBody), &response)
				require.NoError(t, err)

				require.Equal(t, tt.actorID, response.Data.Id)
				require.Equal(t, "actor1", response.Data.Name)
				require.NotZero(t, response.Data.CreatedAt)

			case http.StatusNotFound:
				require.Empty(t, resBody)

			case http.StatusUnauthorized:
				if tt.expectedBody != "" {
					resJson := testhelper.JsonToMap(t, resBody)
					require.Contains(t, resJson, "error")
					require.Contains(t, resJson["error"], tt.expectedBody)
				}

			default:
				t.Fatalf("Unexpected status code: %d", tt.expectedStatus)
			}
		})
	}
}

func TestAdminUpdateActorEndpoint(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tests := []struct {
		name           string
		actorID        int64
		body           string
		token          string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid admin token and existing actor",
			actorID:        1,
			body:           `{"name":"updated_actor"}`,
			token:          "admin-token",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"data":{"id":1,"name":"updated_actor"}}`,
		},
		{
			name:           "Valid admin token but non-existent actor",
			actorID:        999,
			body:           `{"name":"updated_actor"}`,
			token:          "admin-token",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid body",
			actorID:        1,
			body:           `{"invalid_json"}`,
			token:          "admin-token",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid character",
		},
		{
			name:           "Non-admin token",
			actorID:        1,
			body:           `{"name":"updated_actor"}`,
			token:          "actor-token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "",
		},
		{
			name:           "Invalid token",
			actorID:        1,
			body:           `{"name":"updated_actor"}`,
			token:          "invalid_token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, accessor, _ := SetupHttpTestWithDb(t, ctx)

			actor1 := fixture.InsertActor(t, ctx, accessor.Source(), "actor1")
			actor2 := fixture.InsertActor(t, ctx, accessor.Source(), "actor2")
			fixture.InsertToken(t, ctx, accessor.Source(), "admin-token", actor1.ID, 0, []string{"admin"})
			fixture.InsertToken(t, ctx, accessor.Source(), "actor-token", actor2.ID, 0, []string{"user"})

			resp, resBody := PatchHttp(t, fmt.Sprintf("%s/v1/admin/actors/%d", server.URL, tt.actorID), tt.body, tt.token)
			require.Equal(t, tt.expectedStatus, resp.StatusCode)

			switch tt.expectedStatus {
			case http.StatusOK:
				var response api.AdminUpdateActor200JSONResponse
				err := json.Unmarshal([]byte(resBody), &response)
				require.NoError(t, err)

				require.Equal(t, tt.actorID, response.Data.Id)
				require.Equal(t, "updated_actor", response.Data.Name)
				require.NotZero(t, response.Data.CreatedAt)

				// Verify the actor was actually updated in the database
				updatedActor, err := accessor.Querier().ActorFindById(ctx, accessor.Source(), tt.actorID)
				require.NoError(t, err)
				require.NotNil(t, updatedActor)
				require.Equal(t, "updated_actor", updatedActor.Name)

			case http.StatusNotFound, http.StatusBadRequest, http.StatusUnauthorized:
				if tt.expectedBody != "" {
					resJson := testhelper.JsonToMap(t, resBody)
					require.Contains(t, resJson, "error")
					require.Contains(t, resJson["error"], tt.expectedBody)
				}

			default:
				t.Fatalf("Unexpected status code: %d", tt.expectedStatus)
			}
		})
	}
}

func TestAdminDeleteActorEndpoint(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tests := []struct {
		name           string
		actorID        int64
		token          string
		setupFunc      func(context.Context, *testing.T, dbaccess.Accessor, int64)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid admin token and existing actor",
			actorID:        1,
			token:          "admin-token",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"data":{}}`,
		},
		{
			name:           "Valid admin token but non-existent actor",
			actorID:        999,
			token:          "admin-token",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "",
		},
		{
			name:           "Non-admin token",
			actorID:        1,
			token:          "actor-token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "",
		},
		{
			name:           "Invalid token",
			actorID:        1,
			token:          "invalid_token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "",
		},
		{
			name:    "Actor with associated configs",
			actorID: 1,
			token:   "admin-token",
			setupFunc: func(ctx context.Context, t *testing.T, accessor dbaccess.Accessor, actorID int64) {
				fixture.InsertConfig(t, ctx, accessor.Source(), actorID, map[string]string{"key": "value"})
			},
			expectedStatus: http.StatusConflict,
			expectedBody:   "Actor has associated configs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, accessor, _ := SetupHttpTestWithDb(t, ctx)

			fixture.InsertActor(t, ctx, accessor.Source(), "actor1")
			adminActor := fixture.InsertActor(t, ctx, accessor.Source(), "actor2")
			fixture.InsertToken(t, ctx, accessor.Source(), "admin-token", adminActor.ID, 0, []string{"admin"})

			if tt.setupFunc != nil {
				tt.setupFunc(ctx, t, accessor, tt.actorID)
			}

			resp, resBody := DeleteHttp(t, fmt.Sprintf("%s/v1/admin/actors/%d", server.URL, tt.actorID), tt.token)
			require.Equal(t, tt.expectedStatus, resp.StatusCode)

			switch tt.expectedStatus {
			case http.StatusOK:
				// Verify the actor was actually deleted from the database
				_, err := accessor.Querier().ActorFindById(ctx, accessor.Source(), tt.actorID)
				require.Error(t, err)
				require.IsType(t, err, sql.ErrNoRows)

			case http.StatusNotFound:
				require.Empty(t, resBody)

			case http.StatusConflict:
				require.Empty(t, resBody)

			case http.StatusUnauthorized:
				if tt.expectedBody != "" {
					resJson := testhelper.JsonToMap(t, resBody)
					require.Contains(t, resJson, "error")
					require.Contains(t, resJson["error"], tt.expectedBody)
				}

			default:
				t.Fatalf("Unexpected status code: %d", tt.expectedStatus)
			}
		})
	}
}
