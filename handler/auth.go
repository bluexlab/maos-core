package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"gitlab.com/navyx/ai/maos/maos-core/api"
)

const TokenContextKey = "ContextToken"

type Token struct {
	Id          string
	AgentId     int64
	QueueId     int64
	ExpireAt    int64
	Permissions []string
}

// TokenFetcher is a function that retrieves a token from the database.
// It returns nil without an error if the token is not found.
type TokenFetcher func(ctx context.Context, apiToken string) (*Token, error)

func NewBearerAuthMiddleware(fetcher TokenFetcher, cacheTtl time.Duration) api.StrictMiddlewareFunc {
	tokenCache := NewApiTokenCache(fetcher, cacheTtl)

	return func(f api.StrictHandlerFunc, operationID string) api.StrictHandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request, args interface{}) (interface{}, error) {
			auth := r.Header.Get("Authorization")
			if auth == "" {
				http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
				return nil, nil
			}

			const prefix = "Bearer "
			if !strings.HasPrefix(auth, prefix) {
				http.Error(w, `{"error":"invalid authorization header"}`, http.StatusUnauthorized)
				return nil, nil
			}

			tokenString := auth[len(prefix):]
			token := tokenCache.GetToken(ctx, tokenString)
			if token == nil {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return nil, nil
			}

			newContext := context.WithValue(ctx, TokenContextKey, token)

			// Token is valid, call the next handler
			return f(newContext, w, r, args)
		}
	}
}
