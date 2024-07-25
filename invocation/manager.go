package invocation

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	jsoniter "github.com/json-iterator/go"
	"github.com/samber/lo"
	"gitlab.com/navyx/ai/maos/maos-core/api"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
	"gitlab.com/navyx/ai/maos/maos-core/internal/listener"
	"gitlab.com/navyx/ai/maos/maos-core/internal/notifier"
	"gitlab.com/navyx/ai/maos/maos-core/internal/startstop"
	"gitlab.com/navyx/ai/maos/maos-core/util"
)

const (
	invokeTopic = "maos_invoke"
)

var (
	json           = jsoniter.ConfigCompatibleWithStandardLibrary
	defaultWaitSec = 10
)

func NewManager(logger *slog.Logger, accessor dbaccess.Accessor) *Manager {
	pgListener := listener.NewPgListener(accessor.Pool())
	notifier := notifier.New(logger, pgListener, func(status startstop.Status) {
		logger.Info("Invocation manager notifier status changed", "status", status)
	})
	return &Manager{
		logger:           logger,
		accessor:         accessor,
		notifier:         notifier,
		invokeDispatcher: NewDispatcher[InvokeRequest](),
	}
}

type InvokeRequest struct {
}

type Manager struct {
	logger           *slog.Logger
	accessor         dbaccess.Accessor
	notifier         *notifier.Notifier
	invokeDispatcher *Dispatcher[InvokeRequest]
	invokeSub        *notifier.Subscription
}

func (m *Manager) Start(ctx context.Context) error {
	invokeSub, err := m.notifier.Listen(ctx, invokeTopic, m.handleInvokeNotify)
	if err != nil {
		return err
	}

	err = m.notifier.Start(ctx)
	if err != nil {
		invokeSub.Unlisten(ctx)
		m.notifier.Stop()
		return err
	}

	m.invokeSub = invokeSub
	return nil
}

func (m *Manager) handleInvokeNotify(topic notifier.NotificationTopic, payload string) {
	m.logger.Info("Received invoke notification", "topic", topic, "payload", payload)
	m.invokeDispatcher.Dispatch(payload, &InvokeRequest{})
}

func (m *Manager) Close(ctx context.Context) error {
	if m.invokeSub != nil {
		m.invokeSub.Unlisten(ctx)
	}

	m.invokeDispatcher.Close()
	m.notifier.Stop()

	return nil
}

func (m *Manager) InsertInvocation(ctx context.Context, callerAgentId int64, request api.CreateInvocationAsyncRequestObject) (api.CreateInvocationAsyncResponseObject, error) {
	m.logger.Info("InsertInvocation start", "callerAgentId", callerAgentId, "requestBody", request.Body)

	if len(request.Body.Meta) == 0 {
		return api.CreateInvocationAsync400JSONResponse{
			N400JSONResponse: api.N400JSONResponse{Error: "Meta is required"},
		}, nil
	}

	metadata, err := json.Marshal(request.Body.Meta)
	if err != nil {
		return api.CreateInvocationAsync500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{
				Error: fmt.Sprintf("Failed to marshal metadata. err: %s", err.Error()),
			},
		}, nil
	}

	payload, err := json.Marshal(request.Body.Payload)
	if err != nil {
		return api.CreateInvocationAsync500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: "Failed to marshal payload"},
		}, nil
	}

	invocation, err := m.accessor.Querier().InvocationInsert(ctx, m.accessor.Source(), &dbsqlc.InvocationInsertParams{
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

	// notify invoke topic with the queue id
	queueId := strconv.FormatInt(invocation.QueueID, 10)
	m.accessor.Querier().PgNotifyOne(ctx, m.accessor.Source(), &dbsqlc.PgNotifyOneParams{
		Topic:   invokeTopic,
		Payload: queueId,
	})

	return api.CreateInvocationAsync201JSONResponse{
		Id: strconv.FormatInt(invocation.ID, 10),
	}, nil
}

func (m *Manager) GetNextInvocation(ctx context.Context, callerAgentId int64, queueId int64, request api.GetNextInvocationRequestObject) (api.GetNextInvocationResponseObject, error) {
	m.logger.Info("GetNextInvocation start", "callerAgentId", callerAgentId, "requestParams", request.Params)

	getAvailble := func() (*dbsqlc.Invocation, error) {
		invocations, err := m.accessor.Querier().InvocationGetAvailable(ctx, m.accessor.Source(), &dbsqlc.InvocationGetAvailableParams{
			AttemptedBy: callerAgentId,
			QueueID:     queueId,
			Max:         1,
		})
		if len(invocations) > 0 {
			return invocations[0], nil
		}
		return nil, err
	}

	waitSec := util.Clamp(*lo.CoalesceOrEmpty(request.Params.Wait, &defaultWaitSec), 0, 60)

	dispatchId := strconv.FormatInt(queueId, 10)
	m.invokeDispatcher.Listen(dispatchId)

	invocation, err := getAvailble()
	if err != nil {
		m.logger.Error("Failed to get next invocation", "err", err)
		return createGetNext500Response("Cannot get next invocation: " + err.Error()), nil
	}

	if invocation != nil {
		return m.createGetNextResponse(invocation)
	}

	r, err := m.invokeDispatcher.WaitFor(dispatchId, time.Duration(waitSec)*time.Second)
	if r != nil && err == nil {
		// got notification. query again
		invocation, err := getAvailble()
		if err != nil {
			m.logger.Error("Failed to get next invocation", "err", err)
			return createGetNext500Response("Cannot get next invocation: " + err.Error()), nil
		}

		if invocation != nil {
			return m.createGetNextResponse(invocation)
		}
	}

	return api.GetNextInvocation404Response{}, nil
}

func (m *Manager) createGetNextResponse(invocation *dbsqlc.Invocation) (api.GetNextInvocationResponseObject, error) {
	metadata := make(map[string]interface{})
	err := json.Unmarshal(invocation.Metadata, &metadata)
	if err != nil {
		m.logger.Error("Failed to unmarshal metadata", "err", err)
		return createGetNext500Response("Failed to unmarshal metadata"), nil
	}

	payload := make(map[string]interface{})
	err = json.Unmarshal(invocation.Payload, &payload)
	if err != nil {
		m.logger.Error("Failed to unmarshal payload", "err", err)
		return createGetNext500Response("Failed to unmarshal payload"), nil
	}

	return api.GetNextInvocation200JSONResponse{
		Id:      strconv.FormatInt(invocation.ID, 10),
		Meta:    metadata,
		Payload: payload,
	}, nil
}

func createGetNext500Response(err string) api.GetNextInvocationResponseObject {
	return api.GetNextInvocation500JSONResponse{
		N500JSONResponse: api.N500JSONResponse{Error: err},
	}
}
