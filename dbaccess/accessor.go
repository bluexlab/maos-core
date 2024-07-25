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

// New returns a new DBAccess PgAccessor.
//
// It takes a pgxpool.Pool configured for the client's Schema.
// The pool must remain open while the core is running.
func New(pool *pgxpool.Pool) Accessor {
	return &PgAccessor{pool, dbsqlc.New()}
}

// NewWithQuerier takes data sources and querier and returns a new DBAccess PgAccessor.
// It is used for testing purposes.
func NewWithQuerier(pool *pgxpool.Pool, querier dbsqlc.Querier) *PgAccessor {
	return &PgAccessor{pool, querier}
}

type Accessor = *PgAccessor
type TxAccessor = *PgTxAccessor

type PgAccessor struct {
	pool    *pgxpool.Pool
	querier dbsqlc.Querier
}

type PgTxAccessor struct {
	PgAccessor
	tx pgx.Tx
}

func (e *PgAccessor) Querier() dbsqlc.Querier {
	return e.querier
}

func (e *PgAccessor) Source() DataSource {
	return e.pool
}

func (e *PgAccessor) Pool() *pgxpool.Pool {
	return e.pool
}

func (e *PgAccessor) Exec(ctx context.Context, sql string) (struct{}, error) {
	_, err := e.Source().Exec(ctx, sql)
	return struct{}{}, err
}

func (t *PgTxAccessor) Source() DataSource {
	return t.tx
}

func (t *PgTxAccessor) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

func (t *PgTxAccessor) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

func (e *PgAccessor) Begin(ctx context.Context) (TxAccessor, error) {
	tx, err := e.Source().Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &PgTxAccessor{PgAccessor: *e, tx: tx}, nil
}
