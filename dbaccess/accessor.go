package dbaccess

import (
	"context"

	"github.com/jackc/pgx/v5"
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
func New(source DataSource) Accessor {
	return &PgAccessor{source, dbsqlc.New()}
}

// NewWithQuerier takes data sources and querier and returns a new DBAccess PgAccessor.
// It is used for testing purposes.
func NewWithQuerier(source DataSource, querier dbsqlc.Querier) *PgAccessor {
	return &PgAccessor{source, querier}
}

type Accessor = *PgAccessor
type TxAccessor = *PgTxAccessor

type PgAccessor struct {
	source  DataSource
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
	return e.source
}

func (e *PgAccessor) Exec(ctx context.Context, sql string) (struct{}, error) {
	_, err := e.source.Exec(ctx, sql)
	return struct{}{}, err
}

func (t *PgTxAccessor) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

func (t *PgTxAccessor) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

func (e *PgAccessor) Begin(ctx context.Context) (TxAccessor, error) {
	tx, err := e.source.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &PgTxAccessor{PgAccessor: PgAccessor{tx, e.querier}, tx: tx}, nil
}
