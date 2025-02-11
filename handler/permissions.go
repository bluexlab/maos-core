package handler

import (
	"context"

	"github.com/samber/lo"
	"gitlab.com/navyx/ai/maos/maos-core/middleware"
)

var (
	// Permissions is a map of operation id to the permissions they require.
	Permissions = map[string][]string{
		"CreateInvocationAsync":          {"create:invocation"},
		"CreateInvocationSync":           {"create:invocation"},
		"GetNextInvocation":              {"read:invocation"},
		"ReturnInvocationResponse":       {"read:invocation"},
		"ListEmbeddingModels":            {"read:completion"},
		"CreateCompletion":               {"create:completion"},
		"AdminListActors":                {"admin"},
		"AdminGetActors":                 {"admin"},
		"AdminCreateActor":               {"admin"},
		"AdminUpdateActor":               {"admin"},
		"AdminDeleteActor":               {"admin"},
		"AdminGetActorConfig":            {"admin"},
		"AdminListApiTokens":             {"admin"},
		"AdminCreateApiToken":            {"admin"},
		"AdminDeleteApiToken":            {"admin"},
		"AdminUpdateConfig":              {"admin"},
		"AdminListDeployments":           {"admin"},
		"AdminGetDeployment":             {"admin"},
		"AdminGetDeploymentResult":       {"admin"},
		"AdminCreateDeployment":          {"admin"},
		"AdminUpdateDeployment":          {"admin"},
		"AdminDeleteDeployment":          {"admin"},
		"AdminSubmitDeployment":          {"admin"},
		"AdminPublishDeployment":         {"admin"},
		"AdminRejectDeployment":          {"admin"},
		"AdminRestartDeployment":         {"admin"},
		"AdminListPodMetrics":            {"admin"},
		"AdminListReferenceConfigSuites": {"admin"},
		"AdminSyncReferenceConfigSuites": {"admin"},
		"AdminGetSetting":                {"admin"},
		"AdminUpdateSetting":             {"admin"},
		"AdminListSecrets":               {"admin"},
		"AdminUpdateSecret":              {"admin"},
		"AdminDeleteSecret":              {"admin"},
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
