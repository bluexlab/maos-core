package invocation

import (
	"context"
	"crypto/rand"
	"encoding/base64"
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
	invokeTopic   = "maos_invoke"
	responseTopic = "maos_response"
)

var (
	json              = jsoniter.ConfigCompatibleWithStandardLibrary
	defaultWaitSec    = 10
	finalizedStatuses = []dbsqlc.InvocationState{
		dbsqlc.InvocationStateCancelled,
		dbsqlc.InvocationStateCompleted,
		dbsqlc.InvocationStateDiscarded,
	}
	querier = dbsqlc.New()
)

func NewManager(logger *slog.Logger, pool dbaccess.SourcePool) *Manager {
	pgListener := listener.NewPgListener(pool)
	notifier := notifier.New(logger, pgListener, func(status startstop.Status) {
		logger.Info("Invocation manager notifier status changed", "status", status)
	})
	return &Manager{
		logger:           logger,
		dataSource:       pool,
		notifier:         notifier,
		invokeDispatcher: NewDispatcher[InvokeRequest](),
	}
}

type InvokeRequest struct {
}

type Manager struct {
	logger           *slog.Logger
	dataSource       dbaccess.DataSource
	notifier         *notifier.Notifier
	invokeDispatcher *Dispatcher[InvokeRequest]
	invokeSub        *notifier.Subscription
	responseSub      *notifier.Subscription
}

func (m *Manager) Start(ctx context.Context) error {
	invokeSub, err := m.notifier.Listen(ctx, invokeTopic, m.handleInvokeNotify)
	if err != nil {
		invokeSub.Unlisten(ctx)
		return err
	}

	// subscribe response topic to keep notifier listening to the topic
	// this is to avoid frequent subscribe/unsubscribe to the topic
	responseSub, _ := m.notifier.Listen(ctx, responseTopic, func(topic notifier.NotificationTopic, payload string) {
		// we do nothing here.
	})

	err = m.notifier.Start(ctx)
	if err != nil {
		invokeSub.Unlisten(ctx)
		m.notifier.Stop()
		return err
	}

	m.invokeSub = invokeSub
	m.responseSub = responseSub
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
	if m.responseSub != nil {
		m.responseSub.Unlisten(ctx)
	}

	m.invokeDispatcher.Close()
	m.notifier.Stop()

	return nil
}

func (m *Manager) InsertInvocation(ctx context.Context, callerActorId int64, request api.CreateInvocationAsyncRequestObject) (api.CreateInvocationAsyncResponseObject, error) {
	m.logger.Debug("InsertInvocation start", "traceId", request.Body.Meta["trace_id"], "callerActorId", callerActorId, "requestBody", request.Body)

	if len(request.Body.Meta) == 0 {
		return api.CreateInvocationAsync400JSONResponse{
			N400JSONResponse: api.N400JSONResponse{Error: "Meta is required"},
		}, nil
	}

	// ensure trace_id is set
	traceId := request.Body.Meta["trace_id"]
	if traceId == nil {
		request.Body.Meta["trace_id"] = generateTraceId()
	}

	metadata, err := json.Marshal(request.Body.Meta)
	if err != nil {
		m.logger.Error("Failed to marshal metadata", "err", err)
		return api.CreateInvocationAsync500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: "Failed to marshal metadata"},
		}, nil
	}

	payload, err := json.Marshal(request.Body.Payload)
	if err != nil {
		m.logger.Error("Failed to marshal payload", "err", err)
		return api.CreateInvocationAsync500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: "Failed to marshal payload"},
		}, nil
	}

	invocation, err := querier.InvocationInsert(ctx, m.dataSource, &dbsqlc.InvocationInsertParams{
		ActorName: request.Body.Actor,
		State:     "available",
		Metadata:  metadata,
		Priority:  1,
		Payload:   payload,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return api.CreateInvocationAsync400JSONResponse{
				N400JSONResponse: api.N400JSONResponse{Error: "actor not found"},
			}, nil
		}
		return nil, err
	}

	// notify invoke topic with the queue id
	queueId := strconv.FormatInt(invocation.QueueID, 10)
	querier.PgNotifyOne(ctx, m.dataSource, &dbsqlc.PgNotifyOneParams{
		Topic:   invokeTopic,
		Payload: queueId,
	})

	return api.CreateInvocationAsync201JSONResponse{
		Id: strconv.FormatInt(invocation.ID, 10),
	}, nil
}

func (m *Manager) GetInvocationById(ctx context.Context, callerActorId int64, request api.GetInvocationByIdRequestObject) (api.GetInvocationByIdResponseObject, error) {
	m.logger.Debug("GetInvocationById start", "callerActorId", callerActorId, "id", request.Id, "wait", request.Params.Wait)

	invocationId, err := strconv.ParseInt(request.Id, 10, 64)
	if err != nil {
		return api.GetInvocationById404Response{}, nil
	}

	getInvocation := func() api.GetInvocationByIdResponseObject {
		invocation, err := querier.InvocationFindById(ctx, m.dataSource, invocationId)
		if invocation == nil || err != nil {
			if err == pgx.ErrNoRows {
				return api.GetInvocationById404Response{}
			}
			m.logger.Error("Failed to find invocation", "err", err)
			return api.GetInvocationById500JSONResponse{
				N500JSONResponse: api.N500JSONResponse{Error: "Failed to find invocation"},
			}
		}
		result, err1 := parseJson(invocation.Result)
		errors, err2 := parseJson(invocation.Errors)
		if err1 != nil || err2 != nil {
			m.logger.Error("Failed to parse result", "err1", err, "err2", err2)
			return api.GetInvocationById500JSONResponse{
				N500JSONResponse: api.N500JSONResponse{Error: "Failed to parse result or errors"},
			}
		}

		meta, err := parseJson(invocation.Metadata)
		if err != nil {
			m.logger.Error("Failed to parse metadata", "err", err)
			return api.GetInvocationById500JSONResponse{
				N500JSONResponse: api.N500JSONResponse{Error: "Failed to parse metadata"},
			}
		}

		if lo.Contains(finalizedStatuses, invocation.State) {
			return api.GetInvocationById200JSONResponse{
				Id:          request.Id,
				AttemptedAt: invocation.AttemptedAt,
				FinalizedAt: invocation.FinalizedAt,
				Meta:        *meta,
				State:       api.InvocationState(invocation.State),
				Result:      result,
				Errors:      errors,
			}
		}
		return api.GetInvocationById202JSONResponse{
			Id:          request.Id,
			AttemptedAt: invocation.AttemptedAt,
			FinalizedAt: invocation.FinalizedAt,
			Meta:        *meta,
			State:       api.InvocationState(invocation.State),
			Result:      result,
			Errors:      errors,
		}
	}

	if request.Params.Wait == nil {
		return getInvocation(), nil
	}

	// subscript to response topic before insert invocation
	// we keep 64 buffer size to avoid missing response before we start to drain the channel
	responseCh := make(chan string, 64)
	responseDone := make(chan struct{})
	responseSub, err := m.notifier.Listen(ctx, responseTopic, func(topic notifier.NotificationTopic, payload string) {
		// make sure we don't block the notifier
		m.logger.Info("Received response notification", "topic", topic, "payload", payload)
		select {
		case <-responseDone:
		case responseCh <- payload:
		default:
		}
	})
	defer responseSub.Unlisten(ctx)

	waitSec := util.Clamp(*lo.CoalesceOrEmpty(request.Params.Wait, &defaultWaitSec), 0, 60)
	timeContext, cancel := context.WithTimeout(ctx, time.Duration(waitSec)*time.Second)
	defer cancel()

	for {
		select {
		case <-timeContext.Done():
			return getInvocation(), nil

		case response := <-responseCh:
			if response == request.Id {
				close(responseDone)
				return getInvocation(), nil
			}
		}
	}
}

func (m *Manager) ExecuteInvocationSync(ctx context.Context, callerActorId int64, request api.CreateInvocationSyncRequestObject) (api.CreateInvocationSyncResponseObject, error) {
	m.logger.Info("ExecuteInvocationSync start", "traceId", request.Body.Meta["trace_id"], "callerActorId", callerActorId, "requestBody", request.Body)

	if len(request.Body.Meta) == 0 {
		return api.CreateInvocationSync400JSONResponse{
			N400JSONResponse: api.N400JSONResponse{Error: "Meta is required"},
		}, nil
	}

	// ensure trace_id is set
	traceId := request.Body.Meta["trace_id"]
	if traceId == nil {
		request.Body.Meta["trace_id"] = generateTraceId()
	}

	metadata, err := json.Marshal(request.Body.Meta)
	if err != nil {
		m.logger.Error("Failed to marshal metadata", "err", err)
		return api.CreateInvocationSync500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: "Failed to marshal metadata"},
		}, nil
	}

	payload, err := json.Marshal(request.Body.Payload)
	if err != nil {
		m.logger.Error("Failed to marshal payload", "err", err)
		return api.CreateInvocationSync500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: "Failed to marshal payload"},
		}, nil
	}

	// subscript to response topic before insert invocation
	// we keep 64 buffer size to avoid missing response before we start to drain the channel
	responseCh := make(chan string, 64)
	responseDone := make(chan struct{})
	responseSub, err := m.notifier.Listen(ctx, responseTopic, func(topic notifier.NotificationTopic, payload string) {
		// make sure we don't block the notifier
		m.logger.Info("Received response notification", "topic", topic, "payload", payload)
		select {
		case <-responseDone:
		case responseCh <- payload:
		default:
		}
	})
	defer responseSub.Unlisten(ctx)

	invocation, err := querier.InvocationInsert(ctx, m.dataSource, &dbsqlc.InvocationInsertParams{
		ActorName: request.Body.Actor,
		State:     "available",
		Metadata:  metadata,
		Priority:  1,
		Payload:   payload,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return api.CreateInvocationSync400JSONResponse{
				N400JSONResponse: api.N400JSONResponse{Error: "actor not found"},
			}, nil
		}
		return nil, err
	}

	// notify invoke topic with the queue id
	queueId := strconv.FormatInt(invocation.QueueID, 10)
	querier.PgNotifyOne(ctx, m.dataSource, &dbsqlc.PgNotifyOneParams{
		Topic:   invokeTopic,
		Payload: queueId,
	})

	waitSec := util.Clamp(*lo.CoalesceOrEmpty(request.Params.Wait, &defaultWaitSec), 0, 60)
	timeContext, cancel := context.WithTimeout(ctx, time.Duration(waitSec)*time.Second)
	defer cancel()

	invocationIdStr := strconv.FormatInt(invocation.ID, 10)

	returnInvication := func() (api.CreateInvocationSyncResponseObject, error) {
		latestInvocation, err := querier.InvocationFindById(ctx, m.dataSource, invocation.ID)
		result, err1 := parseJson(latestInvocation.Result)
		errors, err2 := parseJson(latestInvocation.Errors)
		if err1 != nil || err2 != nil {
			m.logger.Error("Failed to parse result", "err1", err, "err2", err2)
			return api.CreateInvocationSync500JSONResponse{
				N500JSONResponse: api.N500JSONResponse{Error: "Failed to parse result or errors"},
			}, nil
		}
		meta, err := parseJson(latestInvocation.Metadata)
		if err != nil {
			m.logger.Error("Failed to parse metadata", "err", err)
			return api.CreateInvocationSync500JSONResponse{
				N500JSONResponse: api.N500JSONResponse{Error: "Failed to parse metadata"},
			}, nil
		}

		return api.CreateInvocationSync201JSONResponse{
			Id:          invocationIdStr,
			AttemptedAt: latestInvocation.AttemptedAt,
			FinalizedAt: latestInvocation.FinalizedAt,
			State:       api.InvocationState(latestInvocation.State),
			Meta:        *meta,
			Result:      result,
			Errors:      errors,
		}, nil
	}

	for {
		select {
		case <-timeContext.Done():
			return returnInvication()

		case response := <-responseCh:
			if response == invocationIdStr {
				close(responseDone)
				return returnInvication()
			}
		}
	}
}

func (m *Manager) ReturnInvocationResponse(ctx context.Context, callerActorId int64, request api.ReturnInvocationResponseRequestObject) (api.ReturnInvocationResponseResponseObject, error) {
	m.logger.Info("ReturnInvocationResponse start", "InvokeId", request.InvokeId, "callerActorId", callerActorId, "requestBody", request.Body.Result)

	// Find the invocation
	invocationId, err := strconv.ParseInt(request.InvokeId, 10, 64)
	if err != nil {
		return api.ReturnInvocationResponse404Response{}, nil
	}

	result, err := json.Marshal(request.Body.Result)
	if err != nil {
		m.logger.Error("Failed to marshal invocation result", "err", err)
		return api.ReturnInvocationResponse500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: "Failed to marshal result"},
		}, nil
	}

	invocation, err := querier.InvocationSetCompleteIfRunning(ctx, m.dataSource, &dbsqlc.InvocationSetCompleteIfRunningParams{
		ID:          invocationId,
		FinalizedAt: time.Now().Unix(),
		FinalizerID: callerActorId,
		Result:      result,
	})

	if err != nil {
		if err == pgx.ErrNoRows {
			return api.ReturnInvocationResponse404Response{}, nil
		}

		m.logger.Error("Failed to set invocation complete", "err", err)
		return api.ReturnInvocationResponse500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: "Failed to set invocation compelte"},
		}, nil
	}

	if invocation == nil {
		return api.ReturnInvocationResponse404Response{}, nil
	}

	// notify response topic with the invocation id
	m.logger.Debug("Notify response topic", "topic", responseTopic, "payload", request.InvokeId)
	querier.PgNotifyOne(ctx, m.dataSource, &dbsqlc.PgNotifyOneParams{
		Topic:   responseTopic,
		Payload: request.InvokeId,
	})

	return api.ReturnInvocationResponse200Response{}, nil
}

func (m *Manager) ReturnInvocationError(ctx context.Context, callerActorId int64, request api.ReturnInvocationErrorRequestObject) (api.ReturnInvocationErrorResponseObject, error) {
	m.logger.Info("ReturnInvocationError start", "InvokeId", request.InvokeId, "callerActorId", callerActorId, "requestBody", request.Body.Errors)

	// Find the invocation
	invocationId, err := strconv.ParseInt(request.InvokeId, 10, 64)
	if err != nil {
		return api.ReturnInvocationError404Response{}, nil
	}

	errors, err := json.Marshal(request.Body.Errors)
	if err != nil {
		m.logger.Error("Failed to marshal invocation result", "err", err)
		return api.ReturnInvocationError500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: "Failed to marshal result"},
		}, nil
	}

	invocation, err := querier.InvocationSetFailureIfRunning(ctx, m.dataSource, &dbsqlc.InvocationSetFailureIfRunningParams{
		ID:          invocationId,
		FinalizedAt: time.Now().Unix(),
		FinalizerID: callerActorId,
		Errors:      errors,
	})

	if err != nil {
		if err == pgx.ErrNoRows {
			return api.ReturnInvocationError404Response{}, nil
		}

		m.logger.Error("Failed to set invocation failure", "err", err)
		return api.ReturnInvocationError500JSONResponse{
			N500JSONResponse: api.N500JSONResponse{Error: "Failed to set invocation failure"},
		}, nil
	}

	if invocation == nil {
		return api.ReturnInvocationError404Response{}, nil
	}

	// notify response topic with the invocation id
	m.logger.Debug("Notify response topic", "topic", responseTopic, "payload", request.InvokeId)
	querier.PgNotifyOne(ctx, m.dataSource, &dbsqlc.PgNotifyOneParams{
		Topic:   responseTopic,
		Payload: request.InvokeId,
	})

	return api.ReturnInvocationError200Response{}, nil
}

func (m *Manager) GetNextInvocation(ctx context.Context, callerActorId int64, queueId int64, request api.GetNextInvocationRequestObject) (api.GetNextInvocationResponseObject, error) {
	m.logger.Info("GetNextInvocation start", "callerActorId", callerActorId, "queueId", queueId, "wait", request.Params.Wait)

	getAvailble := func() (*dbsqlc.Invocation, error) {
		invocations, err := querier.InvocationGetAvailable(ctx, m.dataSource, &dbsqlc.InvocationGetAvailableParams{
			AttemptedBy: callerActorId,
			QueueID:     queueId,
			Max:         1,
		})
		if len(invocations) > 0 {
			return invocations[0], nil
		}
		return nil, err
	}

	waitSec := util.Clamp(*lo.CoalesceOrEmpty(request.Params.Wait, &defaultWaitSec), 1, 60)

	dispatchId := strconv.FormatInt(queueId, 10)
	m.invokeDispatcher.Listen(dispatchId)

	startTime := time.Now().Unix()
	for {
		invocation, err := getAvailble()
		if err != nil {
			m.logger.Error("Failed to get next invocation", "err", err)
			return createGetNext500Response("Cannot get next invocation: " + err.Error()), nil
		}

		if invocation != nil {
			m.logger.Debug("Return NextInvocation", "InvokeId", invocation.ID)
			return m.createGetNextResponse(invocation)
		}

		remainingSec := waitSec - int(time.Now().Unix()-startTime)
		if remainingSec <= 0 {
			break
		}

		_, err = m.invokeDispatcher.WaitFor(dispatchId, time.Duration(remainingSec)*time.Second)
		if err != nil {
			m.logger.Error("Failed to wait for next invocation", "err", err)
			return createGetNext500Response("Cannot wait for next invocation:"), nil
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

func parseJson(data []byte) (*map[string]interface{}, error) {
	if data == nil {
		return nil, nil
	}
	var result map[string]interface{}
	err := json.Unmarshal(data, &result)
	return &result, err
}

func generateTraceId() string {
	numBytes := (32*3)/4 + 1

	randomBytes := make([]byte, numBytes)
	_, err := rand.Read(randomBytes)
	if err != nil {
		panic(fmt.Errorf("failed to generate random bytes: %v", err))
	}

	return base64.RawURLEncoding.EncodeToString(randomBytes)
}
