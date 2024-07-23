package invocation

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
	jsoniter "github.com/json-iterator/go"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

func InsertInvocation(ctx context.Context, accessor dbaccess.Accessor, callerAgentId int64, request api.CreateInvocationAsyncRequestObject) (api.CreateInvocationAsyncResponseObject, error) {
	metadata, err := json.Marshal(request.Body.Meta)
	if err != nil {
		return api.CreateInvocationAsync500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{
				Error: fmt.Sprintf("failed to marshal metadata. err: %s", err.Error()),
			},
		}, nil
	}

	payload, err := json.Marshal(request.Body.Payload)
	if err != nil {
		return api.CreateInvocationAsync500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: "failed to marshal payload"},
		}, err
	}

	invocationId, err := accessor.Querier().InvocationInsert(ctx, accessor.Source(), &dbsqlc.InvocationInsertParams{
		AgentName: request.Body.Agent,
		State:     "available",
		Metadata:  metadata,
		Priority:  1,
		Payload:   payload,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return api.CreateInvocationAsync400JSONResponse{
				N400JSONResponse: api.N400JSONResponse{Error: "agent not found"},
			}, nil
		}
		return nil, err
	}

	return api.CreateInvocationAsync201JSONResponse{
		Id: strconv.FormatInt(invocationId, 10),
	}, nil
}
