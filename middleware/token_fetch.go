package middleware

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
)

var (
	bootstrapping = true
	querier       = dbsqlc.New()
)

// NewDatabaseApiTokenFetch creates a TokenFetcher that retrieves tokens from the database.
// It uses the provided accessor to perform database queries.
//
// The bootstrapApiToken is used during system initialization to create the first actor and API token.
// This bootstrap token is disregarded once the first API token has been created in the system.
//
// Parameters:
//   - accessor: The database accessor used for querying tokens.
//   - bootstrapApiToken: The initial API token used for system bootstrapping.
//
// Returns:
//   - A TokenFetcher instance that fetches tokens from the database.
func NewDatabaseApiTokenFetch(dataSource dbaccess.DataSource, bootstrapApiToken string) TokenFetcher {
	return func(ctx context.Context, apiToken string) (*Token, error) {
		if bootstrapping {
			count, err := querier.ApiTokenCount(ctx, dataSource)
			if err != nil {
				return nil, err
			}
			if count == 0 && bootstrapApiToken != "" && apiToken == bootstrapApiToken {
				return &Token{
					Id:          "bootstraping",
					ActorId:     0,
					QueueId:     0,
					ExpireAt:    time.Now().Add(1 * time.Minute).Unix(),
					Permissions: []string{"admin"},
				}, nil
			}
			bootstrapping = count == 0
		}

		token, err := querier.ApiTokenFindByID(ctx, dataSource, apiToken)
		if err != nil {
			if err == pgx.ErrNoRows {
				return nil, nil
			}
			return nil, err
		}
		return &Token{
			Id:          token.ID,
			ActorId:     token.ActorId,
			QueueId:     token.QueueID,
			ExpireAt:    token.ExpireAt,
			Permissions: token.Permissions,
		}, nil
	}
}
