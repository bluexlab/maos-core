package apitest

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/navyx/ai/maos/maos-core/internal/fixture"
)

func TestAdminSyncReferenceConfigSuites(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("Successful sync", func(t *testing.T) {
		server, ds, _, suiteStore := SetupHttpTestWithDbAndSuiteStore(t, ctx)

		actor := fixture.InsertActor(t, ctx, ds, "actor1")
		fixture.InsertToken(t, ctx, ds, "admin-token", actor.ID, 0, []string{"admin"})

		resp, _ := PostHttp(t, server.URL+"/v1/admin/reference_config_suites/sync", "", "admin-token")
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		// Verify that SyncSuites was called
		require.True(t, suiteStore.IsSynced())

		// Add more specific assertions if needed
	})

	t.Run("Unauthorized access", func(t *testing.T) {
		server, _, _ := SetupHttpTestWithDb(t, ctx)

		resp, _ := PostHttp(t, server.URL+"/v1/admin/reference_config_suites/sync", "", "invalid-token")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Non-admin access", func(t *testing.T) {
		server, ds, _ := SetupHttpTestWithDb(t, ctx)

		actor := fixture.InsertActor(t, ctx, ds, "non-admin")
		fixture.InsertToken(t, ctx, ds, "non-admin-token", actor.ID, 0, []string{"user"})

		resp, _ := PostHttp(t, server.URL+"/v1/admin/reference_config_suites/sync", "", "non-admin-token")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	// Add more test cases as needed, such as:
	// - Testing with an empty database
	// - Testing with various error conditions (e.g., database errors)
	// - Testing idempotency (running sync multiple times)
}
