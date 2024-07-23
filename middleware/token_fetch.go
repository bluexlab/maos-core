package middleware

import (
	"context"

	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
)

func NewDatabaseApiTokenFetch(accessor dbaccess.Accessor) TokenFetcher {
	return func(ctx context.Context, apiToken string) (*Token, error) {
		token, err := accessor.Querier().ApiTokenFindByID(ctx, accessor.Source(), apiToken)
		if err != nil {
			return nil, err
		}
		return &Token{
			Id:          token.ID,
			AgentId:     token.AgentID,
			QueueId:     token.QueueID,
			ExpireAt:    token.ExpireAt,
			Permissions: token.Permissions,
		}, nil
	}
}
