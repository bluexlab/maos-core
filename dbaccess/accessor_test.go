package dbaccess

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
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

func TestInterpretError(t *testing.T) {
	t.Parallel()

	require.EqualError(t, interpretError(errors.New("an error")), "an error")
	require.ErrorIs(t, interpretError(pgx.ErrNoRows), ErrNotFound)
	require.NoError(t, interpretError(nil))
}
