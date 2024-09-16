package admin

import (
	"context"
	"fmt"
	"log/slog"

	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/k8s"
)

func ListSecrets(ctx context.Context, k8sController k8s.Controller) (api.AdminListSecretsResponseObject, error) {
	slog.Info("Listing secrets")

	secrets, err := k8sController.ListSecrets(ctx)
	if err != nil {
		return &api.AdminListSecrets500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{
				Error: fmt.Sprintf("Failed to list secrets: %v", err),
			},
		}, nil
	}

	responseSecrets := make([]struct {
		Keys []string `json:"keys"`
		Name string   `json:"name"`
	}, len(secrets))
	for i, secret := range secrets {
		responseSecrets[i] = struct {
			Keys []string `json:"keys"`
			Name string   `json:"name"`
		}{
			Name: secret.Name,
			Keys: secret.Keys,
		}
	}

	return &api.AdminListSecrets200JSONResponse{
		Data: responseSecrets,
	}, nil
}

func UpdateSecret(ctx context.Context, k8sController k8s.Controller, request api.AdminUpdateSecretRequestObject) (api.AdminUpdateSecretResponseObject, error) {
	slog.Info("Updating secret", "name", request.Name)

	secretName := request.Name
	secretData := make(map[string]string)
	for key, value := range *request.Body {
		secretData[key] = value
	}

	err := k8sController.UpdateSecret(ctx, secretName, secretData)
	if err != nil {
		return &api.AdminUpdateSecret500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{
				Error: fmt.Sprintf("Failed to update secret: %v", err),
			},
		}, nil
	}

	return &api.AdminUpdateSecret200Response{}, nil
}

func DeleteSecret(ctx context.Context, k8sController k8s.Controller, request api.AdminDeleteSecretRequestObject) (api.AdminDeleteSecretResponseObject, error) {
	slog.Info("Deleting secret", "name", request.Name)

	err := k8sController.DeleteSecret(ctx, request.Name)
	if err != nil {
		return &api.AdminDeleteSecret500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{
				Error: fmt.Sprintf("Failed to delete secret: %v", err),
			},
		}, nil
	}

	return &api.AdminDeleteSecret200Response{}, nil
}
