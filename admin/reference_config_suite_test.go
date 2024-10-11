package admin_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/navyx/ai/maos/maos-core/admin"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/internal/testhelper"
)

func TestListReferenceConfigSuites(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := testhelper.Logger(t)

	t.Run("Successful listing", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)

		// Insert test data
		testSuites := []struct {
			name    string
			content []byte
		}{
			{"suite1", []byte(`[{"actor_name": "actor1", "configs": {"key1": "value1"}}, {"actor_name": "actor2", "configs": {"key2": "value2"}}]`)},
			{"suite2", []byte(`[{"actor_name": "actor1", "configs": {"key3": "value3"}}, {"actor_name": "actor2", "configs": {"key4": "value4"}}]`)},
		}

		for _, suite := range testSuites {
			_, err := dbPool.Exec(ctx, "INSERT INTO reference_config_suites (name, config_suites) VALUES ($1, $2)", suite.name, suite.content)
			require.NoError(t, err)
		}

		request := api.AdminListReferenceConfigSuitesRequestObject{}
		response, err := admin.ListReferenceConfigSuites(ctx, logger, dbPool, request)

		require.NoError(t, err)
		require.IsType(t, api.AdminListReferenceConfigSuites200JSONResponse{}, response)

		jsonResponse := response.(api.AdminListReferenceConfigSuites200JSONResponse)
		require.NotNil(t, jsonResponse.Data)
		require.Len(t, jsonResponse.Data, 2)

		expectedActorSuites := map[string]api.ReferenceConfigSuite{
			"actor1": {
				ActorName: "actor1",
				ConfigSuites: []struct {
					Configs   map[string]string `json:"configs"`
					SuiteName string            `json:"suite_name"`
				}{
					{SuiteName: "suite1", Configs: map[string]string{"key1": "value1"}},
					{SuiteName: "suite2", Configs: map[string]string{"key3": "value3"}},
				},
			},
			"actor2": {
				ActorName: "actor2",
				ConfigSuites: []struct {
					Configs   map[string]string `json:"configs"`
					SuiteName string            `json:"suite_name"`
				}{
					{SuiteName: "suite1", Configs: map[string]string{"key2": "value2"}},
					{SuiteName: "suite2", Configs: map[string]string{"key4": "value4"}},
				},
			},
		}

		for _, suite := range jsonResponse.Data {
			expectedSuite, exists := expectedActorSuites[suite.ActorName]
			require.True(t, exists)
			require.Equal(t, expectedSuite.ActorName, suite.ActorName)
			require.ElementsMatch(t, expectedSuite.ConfigSuites, suite.ConfigSuites)
		}
	})

	t.Run("Empty list", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)

		request := api.AdminListReferenceConfigSuitesRequestObject{}
		response, err := admin.ListReferenceConfigSuites(ctx, logger, dbPool, request)

		require.NoError(t, err)
		require.IsType(t, api.AdminListReferenceConfigSuites200JSONResponse{}, response)

		jsonResponse := response.(api.AdminListReferenceConfigSuites200JSONResponse)
		require.NotNil(t, jsonResponse.Data)
		require.Len(t, jsonResponse.Data, 0)
	})

	t.Run("Database error", func(t *testing.T) {
		t.Parallel()
		dbPool := testhelper.TestDB(ctx, t)

		// Close the database pool to simulate a database error
		dbPool.Close()

		request := api.AdminListReferenceConfigSuitesRequestObject{}
		response, err := admin.ListReferenceConfigSuites(ctx, logger, dbPool, request)

		require.NoError(t, err)
		require.IsType(t, api.AdminListReferenceConfigSuites500JSONResponse{}, response)

		errorResponse := response.(api.AdminListReferenceConfigSuites500JSONResponse)
		require.Contains(t, errorResponse.Error, "Cannot list reference config suites")
	})
}
