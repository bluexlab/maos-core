package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/samber/lo"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/internal/suitestore"
)

func ListReferenceConfigSuites(ctx context.Context, logger *slog.Logger, accessor dbaccess.Accessor, request api.AdminListReferenceConfigSuitesRequestObject) (api.AdminListReferenceConfigSuitesResponseObject, error) {
	logger.Info("ListReferenceConfigSuites")
	suites, err := accessor.Querier().ReferenceConfigSuiteList(ctx, accessor.Source())
	if err != nil {
		return api.AdminListReferenceConfigSuites500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot list reference config suites: %v", err)},
		}, nil
	}

	actorSuites := make(map[string]api.ReferenceConfigSuite)
	for _, suite := range suites {
		var content []struct {
			ActorName string            `json:"actor_name"`
			Configs   map[string]string `json:"configs"`
		}

		err = json.Unmarshal(suite.ConfigSuite, &content)
		if err != nil {
			logger.Error("Cannot unmarshal reference config suite", "error", err)
			continue
		}

		for _, cont := range content {
			actorSuites[cont.ActorName] = api.ReferenceConfigSuite{
				ActorName: cont.ActorName,
				ConfigSuites: append(actorSuites[cont.ActorName].ConfigSuites, struct {
					Configs   map[string]string `json:"configs"`
					SuiteName string            `json:"suite_name"`
				}{
					SuiteName: suite.Name,
					Configs:   cont.Configs,
				}),
			}
		}
	}

	jsonData, err := json.Marshal(actorSuites)
	logger.Info("ListReferenceConfigSuites", "data", string(jsonData), "error", err)
	return api.AdminListReferenceConfigSuites200JSONResponse{
		Data: lo.Values(actorSuites),
	}, nil
}

func SyncReferenceConfigSuites(ctx context.Context, logger *slog.Logger, suitestore suitestore.SuiteStore) (api.AdminSyncReferenceConfigSuitesResponseObject, error) {
	logger.Info("SyncReferenceConfigSuites")
	err := suitestore.SyncSuites(ctx)
	if err != nil {
		return api.AdminSyncReferenceConfigSuites500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot sync reference config suites: %v", err)},
		}, nil
	}
	return api.AdminSyncReferenceConfigSuites201Response{}, nil
}

func return500Error(logger *slog.Logger, logMessage string, err error) (api.AdminListReferenceConfigSuitesResponseObject, error) {
	logger.Error(logMessage, "error", err)
	return api.AdminListReferenceConfigSuites500JSONResponse{
		N500JSONResponse: api.N500JSONResponse{Error: fmt.Sprintf("Cannot list reference config suites: %v", err)},
	}, nil
}
