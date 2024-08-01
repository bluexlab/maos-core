package middleware

import (
	"context"
	"log/slog"
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
type CacheCloser func()

func NewBearerAuthMiddleware(fetcher TokenFetcher, cacheTtl time.Duration) (api.StrictMiddlewareFunc, CacheCloser) {
	tokenCache := NewApiTokenCache(fetcher, cacheTtl)
	closer := func() {
		tokenCache.cache.Close()
	}

	return func(f api.StrictHandlerFunc, operationID string) api.StrictHandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request, args interface{}) (interface{}, error) {
			if r != nil {
				slog.Debug("BearerAuthMiddleware", "request-uri", r.URL.RequestURI(), "method", r.Method, "remote", r.RemoteAddr, "operationID", operationID, "token", maskAuthToken(r.Header.Get("Authorization")))
			} else {
				slog.Debug("BearerAuthMiddleware. request is blank")
				http.Error(w, `{"error":"Blank request"}`, http.StatusBadRequest)
				return nil, nil
			}

			auth := r.Header.Get("Authorization")
			if auth == "" {
				// No token provided
				// call the next handler with no token in context
				return f(ctx, w, r, args)
			}

			const prefix = "Bearer "
			if !strings.HasPrefix(auth, prefix) {
				http.Error(w, `{"error":"Invalid authorization header"}`, http.StatusUnauthorized)
				return nil, nil
			}

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

func maskAuthToken(token string) string {
	token = strings.TrimSpace(token)
	length := len(token)

	if length <= 0 {
		return ""
	} else if length <= 6 {
		return "******"
	}

	last6 := token[length-6:]

	return "******" + last6
}
