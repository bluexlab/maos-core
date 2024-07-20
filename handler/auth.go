package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"gitlab.com/navyx/ai/maos/maos-core/api"
)

type TokenValidator func(ctx context.Context, token string) (context.Context, error)

func NewBearerAuthMiddleware(validator TokenValidator) api.StrictMiddlewareFunc {
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

			token := auth[len(prefix):]
			newContext, err := validator(ctx, token)
			if err != nil {
				errorMessage, _ := json.Marshal(err.Error())
				http.Error(w, fmt.Sprintf(`{"error":%s}`, errorMessage), http.StatusUnauthorized)
				return nil, nil
			}

			// Token is valid, call the next handler
			return f(newContext, w, r, args)
		}
	}
}
