package dbaccess

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
	"gitlab.com/navyx/ai/maos/maos-core/util"
)

type DataSource interface {
	dbsqlc.DBTX
	Begin(ctx context.Context) (pgx.Tx, error)
}

// New returns a new DBAccess PgAccessor.
//
// It takes a pgxpool.Pool configured for the client's Schema.
// The pool must remain open while the core is running.
func New(source DataSource) *PgAccessor {
	return &PgAccessor{source, dbsqlc.New()}
}

type PgAccessor struct {
	source  DataSource
	queries *dbsqlc.Queries
}

type PgAccessorTx struct {
	PgAccessor
	tx pgx.Tx
}

func (t *PgAccessorTx) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

func (t *PgAccessorTx) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

func (e *PgAccessor) Begin(ctx context.Context) (TxAccessor, error) {
	tx, err := e.source.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &PgAccessorTx{PgAccessor: PgAccessor{tx, e.queries}, tx: tx}, nil
}

func (e *PgAccessor) Exec(ctx context.Context, sql string) (struct{}, error) {
	_, err := e.source.Exec(ctx, sql)
	return struct{}{}, interpretError(err)
}

func (e *PgAccessor) ApiTokenFindByID(ctx context.Context, id string) (*dbsqlc.ApiTokenFindByIDRow, error) {
	token, err := e.queries.ApiTokenFindByID(ctx, e.source, id)
	if err != nil {
		return nil, interpretError(err)
	}
	return token, nil
}

func (e *PgAccessor) MigrationDeleteByVersionMany(ctx context.Context, versions []int) ([]*Migration, error) {
	migrations, err := e.queries.MigrationDeleteByVersionMany(ctx, e.source,
		util.MapSlice(versions, func(v int) int64 { return int64(v) }))
	if err != nil {
		return nil, interpretError(err)
	}
	return util.MapSlice(migrations, migrationFromInternal), nil
}

func (e *PgAccessor) MigrationGetAll(ctx context.Context) ([]*Migration, error) {
	migrations, err := e.queries.MigrationGetAll(ctx, e.source)
	if err != nil {
		return nil, interpretError(err)
	}
	return util.MapSlice(migrations, migrationFromInternal), nil
}

func (e *PgAccessor) MigrationInsertMany(ctx context.Context, versions []int) ([]*Migration, error) {
	migrations, err := e.queries.MigrationInsertMany(ctx, e.source,
		util.MapSlice(versions, func(v int) int64 { return int64(v) }))
	if err != nil {
		return nil, interpretError(err)
	}
	return util.MapSlice(migrations, migrationFromInternal), nil
}

func (e *PgAccessor) TableExists(ctx context.Context, tableName string) (bool, error) {
	exists, err := e.queries.TableExists(ctx, e.source, tableName)
	return exists, interpretError(err)
}

func migrationFromInternal(internal *dbsqlc.Migration) *Migration {
	return &Migration{
		ID:        int(internal.ID),
		CreatedAt: internal.CreatedAt,
		Version:   int(internal.Version),
	}
}

func interpretError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
