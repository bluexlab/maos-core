package middleware

import (
	"context"

	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
)

func NewDatabaseApiTokenFetch(accessor dbaccess.Accessor, sysApiToken string) TokenFetcher {
	return func(ctx context.Context, apiToken string) (*Token, error) {
		if sysApiToken != "" && apiToken == sysApiToken {
			return &Token{
				Id:          "sys",
				AgentId:     0,
				QueueId:     0,
				ExpireAt:    0,
				Permissions: []string{"admin"},
			}, nil
		}

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
