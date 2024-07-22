package dbaccess

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("AllowsNilDatabasePool", func(t *testing.T) {
		t.Parallel()

		dbPool := &pgxpool.Pool{}
		executor := New(dbPool)
		require.Equal(t, dbPool, executor.source)
	})

	t.Run("AllowsNilDatabasePool", func(t *testing.T) {
		t.Parallel()

		executor := New(nil)
		require.Nil(t, executor.source)
	})
}
