package fixture

import (
	"context"

	"github.com/jackc/pgx/v5"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
)

type DataSource interface {
	dbsqlc.DBTX
	Begin(ctx context.Context) (pgx.Tx, error)
}
