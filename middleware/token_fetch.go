package middleware

import (
	"context"

	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
)

var (
	bootstrapping = true
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
func NewDatabaseApiTokenFetch(accessor dbaccess.Accessor, bootstrapApiToken string) TokenFetcher {
	return func(ctx context.Context, apiToken string) (*Token, error) {
		if bootstrapping {
			count, err := accessor.Querier().ApiTokenCount(ctx, accessor.Source())
			if err != nil {
				return nil, err
			}
			if count == 0 && bootstrapApiToken != "" && apiToken == bootstrapApiToken {
				return &Token{
					Id:          "bootstraping",
					ActorId:     0,
					QueueId:     0,
					ExpireAt:    0,
					Permissions: []string{"admin"},
				}, nil
			}
			bootstrapping = count == 0
		}

		token, err := accessor.Querier().ApiTokenFindByID(ctx, accessor.Source(), apiToken)
		if err != nil {
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
