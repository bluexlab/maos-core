package handler

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"
	"gitlab.com/navyx/ai/maos/maos-core/pkg/api"
)

type APIHandler struct{}

// GetCallerConfig implements the GET /v1/config endpoint
func (s *APIHandler) GetCallerConfig(ctx context.Context, request api.GetCallerConfigRequestObject) (api.GetCallerConfigResponseObject, error) {
	config := api.Configuration{
		"key1": "value1",
		"key2": "value2",
	}
	return api.GetCallerConfig200JSONResponse(config), nil
}

// GetNextInvocation implements the GET /v1/invocation/next endpoint
func (s *APIHandler) GetNextInvocation(ctx context.Context, request api.GetNextInvocationRequestObject) (api.GetNextInvocationResponseObject, error) {
	jobId := "job-dummy"
	payload := map[string]interface{}{
		"task": "example_task",
		"data": "example_data",
	}
	job := api.InvocationJob{
		Id:      &jobId,
		Payload: &payload,
	}
	return api.GetNextInvocation200JSONResponse(job), nil
}

// ReturnInvocationResponse implements the POST /v1/invocation/{invoke_id}/response endpoint
func (s *APIHandler) ReturnInvocationResponse(ctx context.Context, request api.ReturnInvocationResponseRequestObject) (api.ReturnInvocationResponseResponseObject, error) {
	// Process the invocation response
	// You can access the invoke_id and result from the request object
	invokeID := request.InvokeId
	result := request.Body.Result

	// Here you would typically store or process the result
	// For this example, we'll just log it
	resultJSON, _ := json.Marshal(result)
	logrus.Infof("Received result for invocation %s: %s", invokeID, string(resultJSON))

	return api.ReturnInvocationResponse200Response{}, nil
}

// ReturnInvocationError implements the POST /v1/invocation/{invoke_id}/error endpoint
func (s *APIHandler) ReturnInvocationError(ctx context.Context, request api.ReturnInvocationErrorRequestObject) (api.ReturnInvocationErrorResponseObject, error) {
	// Process the invocation error
	// You can access the invoke_id and error from the request object
	invokeID := request.InvokeId
	errorDetails := request.Body.Error

	// Here you would typically store or process the error
	// For this example, we'll just log it
	errorJSON, _ := json.Marshal(errorDetails)
	logrus.Infof("Received error for invocation: %s: %s", invokeID, string(errorJSON))

	return api.ReturnInvocationError200Response{}, nil
}
