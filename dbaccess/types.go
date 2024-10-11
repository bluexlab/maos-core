package dbaccess

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
)

type DataSource interface {
	dbsqlc.DBTX
	Begin(ctx context.Context) (pgx.Tx, error)
}

type SourcePool interface {
	dbsqlc.DBTX
	Begin(ctx context.Context) (pgx.Tx, error)
	Acquire(ctx context.Context) (*pgxpool.Conn, error)
}
