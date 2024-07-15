// Package api provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen version v1.16.3 DO NOT EDIT.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/oapi-codegen/runtime"
	strictnethttp "github.com/oapi-codegen/runtime/strictmiddleware/nethttp"
)

const (
	BearerAuthScopes = "bearerAuth.Scopes"
	TraceScopes      = "trace.Scopes"
)

// CreateEmbeddingJSONBody defines parameters for CreateEmbedding.
type CreateEmbeddingJSONBody struct {
	// Input The text to embedded.
	Input []string `json:"input"`

	// ModelId The model id.
	ModelId string `json:"model_id"`
}

// ReturnInvocationErrorJSONBody defines parameters for ReturnInvocationError.
type ReturnInvocationErrorJSONBody struct {
	// Error The error details of the invocation
	Error *map[string]interface{} `json:"error,omitempty"`
}

// ReturnInvocationResponseJSONBody defines parameters for ReturnInvocationResponse.
type ReturnInvocationResponseJSONBody struct {
	// Result The result of the invocation
	Result *map[string]interface{} `json:"result,omitempty"`
}

// CreateEmbeddingJSONRequestBody defines body for CreateEmbedding for application/json ContentType.
type CreateEmbeddingJSONRequestBody CreateEmbeddingJSONBody

// ReturnInvocationErrorJSONRequestBody defines body for ReturnInvocationError for application/json ContentType.
type ReturnInvocationErrorJSONRequestBody ReturnInvocationErrorJSONBody

// ReturnInvocationResponseJSONRequestBody defines body for ReturnInvocationResponse for application/json ContentType.
type ReturnInvocationResponseJSONRequestBody ReturnInvocationResponseJSONBody

// ServerInterface represents all server handlers.
type ServerInterface interface {
	// Get configuration of the caller
	// (GET /v1/config)
	GetCallerConfig(w http.ResponseWriter, r *http.Request)
	// Create embedding of text.
	// (POST /v1/embedding)
	CreateEmbedding(w http.ResponseWriter, r *http.Request)
	// List embedding models.
	// (GET /v1/embedding/models)
	ListEmbeddingModels(w http.ResponseWriter, r *http.Request)
	// Get next invocation job
	// (GET /v1/invocations/next)
	GetNextInvocation(w http.ResponseWriter, r *http.Request)
	// Return invocation error
	// (POST /v1/invocations/{invoke_id}/error)
	ReturnInvocationError(w http.ResponseWriter, r *http.Request, invokeId string)
	// Return invocation result
	// (POST /v1/invocations/{invoke_id}/response)
	ReturnInvocationResponse(w http.ResponseWriter, r *http.Request, invokeId string)
}

// ServerInterfaceWrapper converts contexts to parameters.
type ServerInterfaceWrapper struct {
	Handler            ServerInterface
	HandlerMiddlewares []MiddlewareFunc
	ErrorHandlerFunc   func(w http.ResponseWriter, r *http.Request, err error)
}

type MiddlewareFunc func(http.Handler) http.Handler

// GetCallerConfig operation middleware
func (siw *ServerInterfaceWrapper) GetCallerConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ctx = context.WithValue(ctx, BearerAuthScopes, []string{})

	ctx = context.WithValue(ctx, TraceScopes, []string{})

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetCallerConfig(w, r)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r.WithContext(ctx))
}

// CreateEmbedding operation middleware
func (siw *ServerInterfaceWrapper) CreateEmbedding(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ctx = context.WithValue(ctx, BearerAuthScopes, []string{})

	ctx = context.WithValue(ctx, TraceScopes, []string{})

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.CreateEmbedding(w, r)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r.WithContext(ctx))
}

// ListEmbeddingModels operation middleware
func (siw *ServerInterfaceWrapper) ListEmbeddingModels(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ctx = context.WithValue(ctx, BearerAuthScopes, []string{})

	ctx = context.WithValue(ctx, TraceScopes, []string{})

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.ListEmbeddingModels(w, r)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r.WithContext(ctx))
}

// GetNextInvocation operation middleware
func (siw *ServerInterfaceWrapper) GetNextInvocation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ctx = context.WithValue(ctx, BearerAuthScopes, []string{})

	ctx = context.WithValue(ctx, TraceScopes, []string{})

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetNextInvocation(w, r)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r.WithContext(ctx))
}

// ReturnInvocationError operation middleware
func (siw *ServerInterfaceWrapper) ReturnInvocationError(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var err error

	// ------------- Path parameter "invoke_id" -------------
	var invokeId string

	err = runtime.BindStyledParameter("simple", false, "invoke_id", mux.Vars(r)["invoke_id"], &invokeId)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "invoke_id", Err: err})
		return
	}

	ctx = context.WithValue(ctx, BearerAuthScopes, []string{})

	ctx = context.WithValue(ctx, TraceScopes, []string{})

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.ReturnInvocationError(w, r, invokeId)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r.WithContext(ctx))
}

// ReturnInvocationResponse operation middleware
func (siw *ServerInterfaceWrapper) ReturnInvocationResponse(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var err error

	// ------------- Path parameter "invoke_id" -------------
	var invokeId string

	err = runtime.BindStyledParameter("simple", false, "invoke_id", mux.Vars(r)["invoke_id"], &invokeId)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "invoke_id", Err: err})
		return
	}

	ctx = context.WithValue(ctx, BearerAuthScopes, []string{})

	ctx = context.WithValue(ctx, TraceScopes, []string{})

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.ReturnInvocationResponse(w, r, invokeId)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r.WithContext(ctx))
}

type UnescapedCookieParamError struct {
	ParamName string
	Err       error
}

func (e *UnescapedCookieParamError) Error() string {
	return fmt.Sprintf("error unescaping cookie parameter '%s'", e.ParamName)
}

func (e *UnescapedCookieParamError) Unwrap() error {
	return e.Err
}

type UnmarshalingParamError struct {
	ParamName string
	Err       error
}

func (e *UnmarshalingParamError) Error() string {
	return fmt.Sprintf("Error unmarshaling parameter %s as JSON: %s", e.ParamName, e.Err.Error())
}

func (e *UnmarshalingParamError) Unwrap() error {
	return e.Err
}

type RequiredParamError struct {
	ParamName string
}

func (e *RequiredParamError) Error() string {
	return fmt.Sprintf("Query argument %s is required, but not found", e.ParamName)
}

type RequiredHeaderError struct {
	ParamName string
	Err       error
}

func (e *RequiredHeaderError) Error() string {
	return fmt.Sprintf("Header parameter %s is required, but not found", e.ParamName)
}

func (e *RequiredHeaderError) Unwrap() error {
	return e.Err
}

type InvalidParamFormatError struct {
	ParamName string
	Err       error
}

func (e *InvalidParamFormatError) Error() string {
	return fmt.Sprintf("Invalid format for parameter %s: %s", e.ParamName, e.Err.Error())
}

func (e *InvalidParamFormatError) Unwrap() error {
	return e.Err
}

type TooManyValuesForParamError struct {
	ParamName string
	Count     int
}

func (e *TooManyValuesForParamError) Error() string {
	return fmt.Sprintf("Expected one value for %s, got %d", e.ParamName, e.Count)
}

// Handler creates http.Handler with routing matching OpenAPI spec.
func Handler(si ServerInterface) http.Handler {
	return HandlerWithOptions(si, GorillaServerOptions{})
}

type GorillaServerOptions struct {
	BaseURL          string
	BaseRouter       *mux.Router
	Middlewares      []MiddlewareFunc
	ErrorHandlerFunc func(w http.ResponseWriter, r *http.Request, err error)
}

// HandlerFromMux creates http.Handler with routing matching OpenAPI spec based on the provided mux.
func HandlerFromMux(si ServerInterface, r *mux.Router) http.Handler {
	return HandlerWithOptions(si, GorillaServerOptions{
		BaseRouter: r,
	})
}

func HandlerFromMuxWithBaseURL(si ServerInterface, r *mux.Router, baseURL string) http.Handler {
	return HandlerWithOptions(si, GorillaServerOptions{
		BaseURL:    baseURL,
		BaseRouter: r,
	})
}

// HandlerWithOptions creates http.Handler with additional options
func HandlerWithOptions(si ServerInterface, options GorillaServerOptions) http.Handler {
	r := options.BaseRouter

	if r == nil {
		r = mux.NewRouter()
	}
	if options.ErrorHandlerFunc == nil {
		options.ErrorHandlerFunc = func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}
	wrapper := ServerInterfaceWrapper{
		Handler:            si,
		HandlerMiddlewares: options.Middlewares,
		ErrorHandlerFunc:   options.ErrorHandlerFunc,
	}

	r.HandleFunc(options.BaseURL+"/v1/config", wrapper.GetCallerConfig).Methods("GET")

	r.HandleFunc(options.BaseURL+"/v1/embedding", wrapper.CreateEmbedding).Methods("POST")

	r.HandleFunc(options.BaseURL+"/v1/embedding/models", wrapper.ListEmbeddingModels).Methods("GET")

	r.HandleFunc(options.BaseURL+"/v1/invocations/next", wrapper.GetNextInvocation).Methods("GET")

	r.HandleFunc(options.BaseURL+"/v1/invocations/{invoke_id}/error", wrapper.ReturnInvocationError).Methods("POST")

	r.HandleFunc(options.BaseURL+"/v1/invocations/{invoke_id}/response", wrapper.ReturnInvocationResponse).Methods("POST")

	return r
}

type GetCallerConfigRequestObject struct {
}

type GetCallerConfigResponseObject interface {
	VisitGetCallerConfigResponse(w http.ResponseWriter) error
}

type GetCallerConfig200JSONResponse map[string]string

func (response GetCallerConfig200JSONResponse) VisitGetCallerConfigResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)

	return json.NewEncoder(w).Encode(response)
}

type GetCallerConfig401Response struct {
}

func (response GetCallerConfig401Response) VisitGetCallerConfigResponse(w http.ResponseWriter) error {
	w.WriteHeader(401)
	return nil
}

type CreateEmbeddingRequestObject struct {
	Body *CreateEmbeddingJSONRequestBody
}

type CreateEmbeddingResponseObject interface {
	VisitCreateEmbeddingResponse(w http.ResponseWriter) error
}

type CreateEmbedding200JSONResponse struct {
	// Data The embeddings of the text.
	Data *[]struct {
		// Embedding The embedding of the text.
		Embedding *[]float32 `json:"embedding,omitempty"`

		// Index The index of the text in the original list.
		Index *int `json:"index,omitempty"`
	} `json:"data,omitempty"`
}

func (response CreateEmbedding200JSONResponse) VisitCreateEmbeddingResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)

	return json.NewEncoder(w).Encode(response)
}

type ListEmbeddingModelsRequestObject struct {
}

type ListEmbeddingModelsResponseObject interface {
	VisitListEmbeddingModelsResponse(w http.ResponseWriter) error
}

type ListEmbeddingModels200JSONResponse struct {
	Data *[]struct {
		// Dimension The dimension of the output vector.
		Dimension *int    `json:"dimension,omitempty"`
		Id        *string `json:"id,omitempty"`
		Name      *string `json:"name,omitempty"`
		Provider  *string `json:"provider,omitempty"`
	} `json:"data,omitempty"`
}

func (response ListEmbeddingModels200JSONResponse) VisitListEmbeddingModelsResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)

	return json.NewEncoder(w).Encode(response)
}

type GetNextInvocationRequestObject struct {
}

type GetNextInvocationResponseObject interface {
	VisitGetNextInvocationResponse(w http.ResponseWriter) error
}

type GetNextInvocation200JSONResponse struct {
	// Id The unique identifier for the invocation job
	Id *string `json:"id,omitempty"`

	// Payload The payload for the invocation job
	Payload *map[string]interface{} `json:"payload,omitempty"`
}

func (response GetNextInvocation200JSONResponse) VisitGetNextInvocationResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)

	return json.NewEncoder(w).Encode(response)
}

type GetNextInvocation401Response struct {
}

func (response GetNextInvocation401Response) VisitGetNextInvocationResponse(w http.ResponseWriter) error {
	w.WriteHeader(401)
	return nil
}

type GetNextInvocation404Response struct {
}

func (response GetNextInvocation404Response) VisitGetNextInvocationResponse(w http.ResponseWriter) error {
	w.WriteHeader(404)
	return nil
}

type ReturnInvocationErrorRequestObject struct {
	InvokeId string `json:"invoke_id"`
	Body     *ReturnInvocationErrorJSONRequestBody
}

type ReturnInvocationErrorResponseObject interface {
	VisitReturnInvocationErrorResponse(w http.ResponseWriter) error
}

type ReturnInvocationError200Response struct {
}

func (response ReturnInvocationError200Response) VisitReturnInvocationErrorResponse(w http.ResponseWriter) error {
	w.WriteHeader(200)
	return nil
}

type ReturnInvocationError401Response struct {
}

func (response ReturnInvocationError401Response) VisitReturnInvocationErrorResponse(w http.ResponseWriter) error {
	w.WriteHeader(401)
	return nil
}

type ReturnInvocationError404Response struct {
}

func (response ReturnInvocationError404Response) VisitReturnInvocationErrorResponse(w http.ResponseWriter) error {
	w.WriteHeader(404)
	return nil
}

type ReturnInvocationResponseRequestObject struct {
	InvokeId string `json:"invoke_id"`
	Body     *ReturnInvocationResponseJSONRequestBody
}

type ReturnInvocationResponseResponseObject interface {
	VisitReturnInvocationResponseResponse(w http.ResponseWriter) error
}

type ReturnInvocationResponse200Response struct {
}

func (response ReturnInvocationResponse200Response) VisitReturnInvocationResponseResponse(w http.ResponseWriter) error {
	w.WriteHeader(200)
	return nil
}

type ReturnInvocationResponse401Response struct {
}

func (response ReturnInvocationResponse401Response) VisitReturnInvocationResponseResponse(w http.ResponseWriter) error {
	w.WriteHeader(401)
	return nil
}

type ReturnInvocationResponse404Response struct {
}

func (response ReturnInvocationResponse404Response) VisitReturnInvocationResponseResponse(w http.ResponseWriter) error {
	w.WriteHeader(404)
	return nil
}

// StrictServerInterface represents all server handlers.
type StrictServerInterface interface {
	// Get configuration of the caller
	// (GET /v1/config)
	GetCallerConfig(ctx context.Context, request GetCallerConfigRequestObject) (GetCallerConfigResponseObject, error)
	// Create embedding of text.
	// (POST /v1/embedding)
	CreateEmbedding(ctx context.Context, request CreateEmbeddingRequestObject) (CreateEmbeddingResponseObject, error)
	// List embedding models.
	// (GET /v1/embedding/models)
	ListEmbeddingModels(ctx context.Context, request ListEmbeddingModelsRequestObject) (ListEmbeddingModelsResponseObject, error)
	// Get next invocation job
	// (GET /v1/invocations/next)
	GetNextInvocation(ctx context.Context, request GetNextInvocationRequestObject) (GetNextInvocationResponseObject, error)
	// Return invocation error
	// (POST /v1/invocations/{invoke_id}/error)
	ReturnInvocationError(ctx context.Context, request ReturnInvocationErrorRequestObject) (ReturnInvocationErrorResponseObject, error)
	// Return invocation result
	// (POST /v1/invocations/{invoke_id}/response)
	ReturnInvocationResponse(ctx context.Context, request ReturnInvocationResponseRequestObject) (ReturnInvocationResponseResponseObject, error)
}

type StrictHandlerFunc = strictnethttp.StrictHttpHandlerFunc
type StrictMiddlewareFunc = strictnethttp.StrictHttpMiddlewareFunc

type StrictHTTPServerOptions struct {
	RequestErrorHandlerFunc  func(w http.ResponseWriter, r *http.Request, err error)
	ResponseErrorHandlerFunc func(w http.ResponseWriter, r *http.Request, err error)
}

func NewStrictHandler(ssi StrictServerInterface, middlewares []StrictMiddlewareFunc) ServerInterface {
	return &strictHandler{ssi: ssi, middlewares: middlewares, options: StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		},
	}}
}

func NewStrictHandlerWithOptions(ssi StrictServerInterface, middlewares []StrictMiddlewareFunc, options StrictHTTPServerOptions) ServerInterface {
	return &strictHandler{ssi: ssi, middlewares: middlewares, options: options}
}

type strictHandler struct {
	ssi         StrictServerInterface
	middlewares []StrictMiddlewareFunc
	options     StrictHTTPServerOptions
}

// GetCallerConfig operation middleware
func (sh *strictHandler) GetCallerConfig(w http.ResponseWriter, r *http.Request) {
	var request GetCallerConfigRequestObject

	handler := func(ctx context.Context, w http.ResponseWriter, r *http.Request, request interface{}) (interface{}, error) {
		return sh.ssi.GetCallerConfig(ctx, request.(GetCallerConfigRequestObject))
	}
	for _, middleware := range sh.middlewares {
		handler = middleware(handler, "GetCallerConfig")
	}

	response, err := handler(r.Context(), w, r, request)

	if err != nil {
		sh.options.ResponseErrorHandlerFunc(w, r, err)
	} else if validResponse, ok := response.(GetCallerConfigResponseObject); ok {
		if err := validResponse.VisitGetCallerConfigResponse(w); err != nil {
			sh.options.ResponseErrorHandlerFunc(w, r, err)
		}
	} else if response != nil {
		sh.options.ResponseErrorHandlerFunc(w, r, fmt.Errorf("unexpected response type: %T", response))
	}
}

// CreateEmbedding operation middleware
func (sh *strictHandler) CreateEmbedding(w http.ResponseWriter, r *http.Request) {
	var request CreateEmbeddingRequestObject

	var body CreateEmbeddingJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sh.options.RequestErrorHandlerFunc(w, r, fmt.Errorf("can't decode JSON body: %w", err))
		return
	}
	request.Body = &body

	handler := func(ctx context.Context, w http.ResponseWriter, r *http.Request, request interface{}) (interface{}, error) {
		return sh.ssi.CreateEmbedding(ctx, request.(CreateEmbeddingRequestObject))
	}
	for _, middleware := range sh.middlewares {
		handler = middleware(handler, "CreateEmbedding")
	}

	response, err := handler(r.Context(), w, r, request)

	if err != nil {
		sh.options.ResponseErrorHandlerFunc(w, r, err)
	} else if validResponse, ok := response.(CreateEmbeddingResponseObject); ok {
		if err := validResponse.VisitCreateEmbeddingResponse(w); err != nil {
			sh.options.ResponseErrorHandlerFunc(w, r, err)
		}
	} else if response != nil {
		sh.options.ResponseErrorHandlerFunc(w, r, fmt.Errorf("unexpected response type: %T", response))
	}
}

// ListEmbeddingModels operation middleware
func (sh *strictHandler) ListEmbeddingModels(w http.ResponseWriter, r *http.Request) {
	var request ListEmbeddingModelsRequestObject

	handler := func(ctx context.Context, w http.ResponseWriter, r *http.Request, request interface{}) (interface{}, error) {
		return sh.ssi.ListEmbeddingModels(ctx, request.(ListEmbeddingModelsRequestObject))
	}
	for _, middleware := range sh.middlewares {
		handler = middleware(handler, "ListEmbeddingModels")
	}

	response, err := handler(r.Context(), w, r, request)

	if err != nil {
		sh.options.ResponseErrorHandlerFunc(w, r, err)
	} else if validResponse, ok := response.(ListEmbeddingModelsResponseObject); ok {
		if err := validResponse.VisitListEmbeddingModelsResponse(w); err != nil {
			sh.options.ResponseErrorHandlerFunc(w, r, err)
		}
	} else if response != nil {
		sh.options.ResponseErrorHandlerFunc(w, r, fmt.Errorf("unexpected response type: %T", response))
	}
}

// GetNextInvocation operation middleware
func (sh *strictHandler) GetNextInvocation(w http.ResponseWriter, r *http.Request) {
	var request GetNextInvocationRequestObject

	handler := func(ctx context.Context, w http.ResponseWriter, r *http.Request, request interface{}) (interface{}, error) {
		return sh.ssi.GetNextInvocation(ctx, request.(GetNextInvocationRequestObject))
	}
	for _, middleware := range sh.middlewares {
		handler = middleware(handler, "GetNextInvocation")
	}

	response, err := handler(r.Context(), w, r, request)

	if err != nil {
		sh.options.ResponseErrorHandlerFunc(w, r, err)
	} else if validResponse, ok := response.(GetNextInvocationResponseObject); ok {
		if err := validResponse.VisitGetNextInvocationResponse(w); err != nil {
			sh.options.ResponseErrorHandlerFunc(w, r, err)
		}
	} else if response != nil {
		sh.options.ResponseErrorHandlerFunc(w, r, fmt.Errorf("unexpected response type: %T", response))
	}
}

// ReturnInvocationError operation middleware
func (sh *strictHandler) ReturnInvocationError(w http.ResponseWriter, r *http.Request, invokeId string) {
	var request ReturnInvocationErrorRequestObject

	request.InvokeId = invokeId

	var body ReturnInvocationErrorJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sh.options.RequestErrorHandlerFunc(w, r, fmt.Errorf("can't decode JSON body: %w", err))
		return
	}
	request.Body = &body

	handler := func(ctx context.Context, w http.ResponseWriter, r *http.Request, request interface{}) (interface{}, error) {
		return sh.ssi.ReturnInvocationError(ctx, request.(ReturnInvocationErrorRequestObject))
	}
	for _, middleware := range sh.middlewares {
		handler = middleware(handler, "ReturnInvocationError")
	}

	response, err := handler(r.Context(), w, r, request)

	if err != nil {
		sh.options.ResponseErrorHandlerFunc(w, r, err)
	} else if validResponse, ok := response.(ReturnInvocationErrorResponseObject); ok {
		if err := validResponse.VisitReturnInvocationErrorResponse(w); err != nil {
			sh.options.ResponseErrorHandlerFunc(w, r, err)
		}
	} else if response != nil {
		sh.options.ResponseErrorHandlerFunc(w, r, fmt.Errorf("unexpected response type: %T", response))
	}
}

// ReturnInvocationResponse operation middleware
func (sh *strictHandler) ReturnInvocationResponse(w http.ResponseWriter, r *http.Request, invokeId string) {
	var request ReturnInvocationResponseRequestObject

	request.InvokeId = invokeId

	var body ReturnInvocationResponseJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sh.options.RequestErrorHandlerFunc(w, r, fmt.Errorf("can't decode JSON body: %w", err))
		return
	}
	request.Body = &body

	handler := func(ctx context.Context, w http.ResponseWriter, r *http.Request, request interface{}) (interface{}, error) {
		return sh.ssi.ReturnInvocationResponse(ctx, request.(ReturnInvocationResponseRequestObject))
	}
	for _, middleware := range sh.middlewares {
		handler = middleware(handler, "ReturnInvocationResponse")
	}

	response, err := handler(r.Context(), w, r, request)

	if err != nil {
		sh.options.ResponseErrorHandlerFunc(w, r, err)
	} else if validResponse, ok := response.(ReturnInvocationResponseResponseObject); ok {
		if err := validResponse.VisitReturnInvocationResponseResponse(w); err != nil {
			sh.options.ResponseErrorHandlerFunc(w, r, err)
		}
	} else if response != nil {
		sh.options.ResponseErrorHandlerFunc(w, r, fmt.Errorf("unexpected response type: %T", response))
	}
}
