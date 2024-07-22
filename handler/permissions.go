package handler

import (
	"context"

	"github.com/samber/lo"
)

var (
	// Permissions is a map of operation id to the permissions they require.
	Permissions = map[string][]string{
		"AdminListApiTokens":  {"admin"},
		"AdminCreateApiToken": {"admin"},
	}
)

func ValidatePermissions(ctx context.Context, operationID string) bool {
	token := ctx.Value(TokenContextKey).(*Token)
	if token == nil {
		return false
	}

	requiredPermissions, ok := Permissions[operationID]
	if !ok {
		return false
	}

	for _, requiredPermission := range requiredPermissions {
		if !lo.Contains(token.Permissions, requiredPermission) {
			return false
		}
	}

	return true
}
