package handler

import (
	"context"

	"github.com/samber/lo"
)

var (
	// Permissions is a map of operation id to the permissions they require.
	Permissions = map[string][]string{
		"AdminListAgents":     {"admin"},
		"AdminCreateAgent":    {"admin"},
		"AdminListApiTokens":  {"admin"},
		"AdminCreateApiToken": {"admin"},
	}
)

func GetContextToken(ctx context.Context) *Token {
	tokenValue := ctx.Value(TokenContextKey)
	if tokenValue == nil {
		return nil
	}

	return tokenValue.(*Token)
}

func ValidatePermissions(ctx context.Context, operationID string) *Token {
	token := GetContextToken(ctx)
	if token == nil {
		return nil
	}

	requiredPermissions, ok := Permissions[operationID]
	if !ok {
		return nil
	}

	for _, requiredPermission := range requiredPermissions {
		if !lo.Contains(token.Permissions, requiredPermission) {
			return nil
		}
	}

	return token
}
