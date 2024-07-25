package handler

import (
	"context"

	"github.com/samber/lo"
	"gitlab.com/navyx/ai/maos/maos-core/middleware"
)

var (
	// Permissions is a map of operation id to the permissions they require.
	Permissions = map[string][]string{
		"CreateInvocationAsync":    {"create:invocation"},
		"CreateInvocationSync":     {"create:invocation"},
		"GetNextInvocation":        {"read:invocation"},
		"ReturnInvocationResponse": {"read:invocation"},
		"AdminListAgents":          {"admin"},
		"AdminCreateAgent":         {"admin"},
		"AdminListApiTokens":       {"admin"},
		"AdminCreateApiToken":      {"admin"},
	}
)

func GetContextToken(ctx context.Context) *middleware.Token {
	tokenValue := ctx.Value(middleware.TokenContextKey)
	if tokenValue == nil {
		return nil
	}

	return tokenValue.(*middleware.Token)
}

func ValidatePermissions(ctx context.Context, operationID string) *middleware.Token {
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
