package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
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
type CacheCloser func()

func NewBearerAuthMiddleware(fetcher TokenFetcher, cacheTtl time.Duration) (api.StrictMiddlewareFunc, CacheCloser) {
	tokenCache := NewApiTokenCache(fetcher, cacheTtl)
	closer := func() {
		tokenCache.cache.Close()
	}

	return func(f api.StrictHandlerFunc, operationID string) api.StrictHandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request, args interface{}) (interface{}, error) {
			auth := r.Header.Get("Authorization")
			if auth == "" {
				http.Error(w, `{"error":"Missing authorization header"}`, http.StatusUnauthorized)
				return nil, nil
			}

			const prefix = "Bearer "
			if !strings.HasPrefix(auth, prefix) {
				http.Error(w, `{"error":"Invalid authorization header"}`, http.StatusUnauthorized)
				return nil, nil
			}

			logrus.Debugf("Authorization token: %s", auth[:6])

			tokenString := auth[len(prefix):]
			token := tokenCache.GetToken(ctx, tokenString)
			if token == nil {
				http.Error(w, `{"error":"Invalid token"}`, http.StatusUnauthorized)
				return nil, nil
			}

			newContext := context.WithValue(ctx, TokenContextKey, token)

			// Token is valid, call the next handler
			return f(newContext, w, r, args)
		}
	}, closer
}
