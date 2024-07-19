package dbaccess

import (
	"context"
)

type Accessor interface {
	// Begin begins a new subtransaction. ErrSubTxNotSupported may be returned
	// if the executor is a transaction and the driver doesn't support
	// subtransactions.
	Begin(ctx context.Context) (TxAccessor, error)

	// Exec executes raw SQL. Used for migrations.
	Exec(ctx context.Context, sql string) (struct{}, error)

	// MigrationDeleteByVersionMany deletes many migration versions.
	MigrationDeleteByVersionMany(ctx context.Context, versions []int) ([]*Migration, error)

	// MigrationGetAll gets all currently applied migrations.
	MigrationGetAll(ctx context.Context) ([]*Migration, error)

	// MigrationInsertMany inserts many migration versions.
	MigrationInsertMany(ctx context.Context, versions []int) ([]*Migration, error)

	TableExists(ctx context.Context, tableName string) (bool, error)
}

type TxAccessor interface {
	Accessor

	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type Migration struct {
	// ID is an automatically generated primary key for the migration.
	//
	// API is not stable. DO NOT USE.
	ID int

	// CreatedAt is when the migration was initially created.
	//
	// API is not stable. DO NOT USE.
	CreatedAt int64

	// Version is the version of the migration.
	//
	// API is not stable. DO NOT USE.
	Version int
}
