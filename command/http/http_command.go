package httpCommand

import (
	"context"
	"fmt"
	"github.com/devlibx/gox-base"
	"github.com/devlibx/gox-base/errors"
	"github.com/devlibx/gox-base/serialization"
	"github.com/devlibx/gox-http/command"
	"github.com/go-resty/resty/v2"
	_ "github.com/go-resty/resty/v2"
	"github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

var EnableGoxHttpMetricLogging = false
var EnableTimeTakenByHttpCall = false

// StartSpanFromContext is added for someone to override the implementation
type StartSpanFromContext func(ctx context.Context, operationName string, opts ...opentracing.StartSpanOption) (opentracing.Span, context.Context)

// DefaultStartSpanFromContextFunc provides a default implementation for StartSpanFromContext function
var DefaultStartSpanFromContextFunc StartSpanFromContext = func(ctx context.Context, operationName string, opts ...opentracing.StartSpanOption) (opentracing.Span, context.Context) {
	return opentracing.StartSpanFromContext(ctx, operationName, opts...)
}

type HttpCommand struct {
	gox.CrossFunction
	server           *command.Server
	api              *command.Api
	logger           *zap.Logger
	client           *resty.Client
	setRetryFuncOnce *sync.Once
}

func (h *HttpCommand) ExecuteAsync(ctx context.Context, request *command.GoxRequest) chan *command.GoxResponse {
	responseChannel := make(chan *command.GoxResponse)
	go func() {
		if result, err := h.Execute(ctx, request); err != nil {
			responseChannel <- &command.GoxResponse{Err: err}
		} else {
			responseChannel <- result
		}
	}()
	return responseChannel
}

func (h *HttpCommand) Execute(ctx context.Context, request *command.GoxRequest) (*command.GoxResponse, error) {

	response, err := h.internalExecute(ctx, request)

	// Log HTTP metrics
	if EnableGoxHttpMetricLogging {
		if err == nil {
			if response == nil {
				h.Metric().Tagged(map[string]string{"server": h.server.Name, "api": h.api.Name, "status": fmt.Sprintf("%d", 200)}).Counter("gox_http_call").Inc(1)
			} else {
				h.Metric().Tagged(map[string]string{"server": h.server.Name, "api": h.api.Name, "status": fmt.Sprintf("%d", response.StatusCode)}).Counter("gox_http_call").Inc(1)
			}
		} else {
			if goxErr, ok := err.(*command.GoxHttpError); ok {
				if response == nil {
					h.Metric().Tagged(map[string]string{"server": h.server.Name, "api": h.api.Name, "status": fmt.Sprintf("%d", 500), "error": goxErr.ErrorCode}).Counter("gox_http_call").Inc(1)
				} else {
					h.Metric().Tagged(map[string]string{"server": h.server.Name, "api": h.api.Name, "status": fmt.Sprintf("%d", response.StatusCode), "error": goxErr.ErrorCode}).Counter("gox_http_call").Inc(1)
				}
			} else {
				if response == nil {
					h.Metric().Tagged(map[string]string{"server": h.server.Name, "api": h.api.Name, "status": fmt.Sprintf("%d", 500), "error": "unknown"}).Counter("gox_http_call").Inc(1)
				} else {
					h.Metric().Tagged(map[string]string{"server": h.server.Name, "api": h.api.Name, "status": fmt.Sprintf("%d", response.StatusCode), "error": "unknown"}).Counter("gox_http_call").Inc(1)
				}
			}
		}
	}

	return response, err
}

func (h *HttpCommand) internalExecute(ctx context.Context, request *command.GoxRequest) (*command.GoxResponse, error) {
	// sp, ctxWithSpan := opentracing.StartSpanFromContext(ctx, h.api.Name)
	sp, ctxWithSpan := DefaultStartSpanFromContextFunc(ctx, h.api.Name)
	defer sp.Finish()

	h.logger.Debug("got request to execute", zap.Stringer("request", request))

	var response *resty.Response

	// Build request with all parameters
	r, err := h.buildRequest(ctxWithSpan, request, sp)
	if err != nil {
		return nil, err
	}

	// Create the url to call
	finalUrlToRequest := h.api.GetPath(h.server)
	h.logger.Debug("url to use", zap.String("url", finalUrlToRequest))

	start := time.Now()
	switch strings.ToUpper(h.api.Method) {
	case "GET":
		response, err = r.Get(finalUrlToRequest)
	case "POST":
		response, err = r.Post(finalUrlToRequest)
	case "PUT":
		response, err = r.Put(finalUrlToRequest)
	case "DELETE":
		response, err = r.Delete(finalUrlToRequest)
	}
	end := time.Now()
	if EnableTimeTakenByHttpCall {
		h.logger.Info("Time taken: ", zap.Int64("time_taken", end.UnixMilli()-start.UnixMilli()), zap.String("url", finalUrlToRequest))
	}

	if err != nil {
		responseObject := h.handleError(err)
		return responseObject, responseObject.Err
	} else {
		responseObject := h.processResponse(request, response)
		return responseObject, responseObject.Err
	}
}

func (h *HttpCommand) buildRequest(ctx context.Context, request *command.GoxRequest, sp opentracing.Span) (*resty.Request, error) {
	r := h.client.R()
	r.SetContext(ctx)

	// If retry is enabled then we will setup retrying
	h.setRetryFuncOnce.Do(func() {
		if h.api.RetryCount >= 0 {

			// Set retry count defined
			h.client.SetRetryCount(h.api.RetryCount)

			// Set initial retry time
			if h.api.InitialRetryWaitTimeMs > 0 {
				h.client.SetRetryWaitTime(time.Duration(h.api.InitialRetryWaitTimeMs) * time.Millisecond)
			}

			// Set retry function to avoid retry if this status is acceptable
			h.client.AddRetryCondition(func(response *resty.Response, err error) bool {
				if response != nil && h.api.IsHttpCodeAcceptable(response.StatusCode()) {
					return false
				}
				if response != nil {
					h.logger.Info("retrying api after error", zap.Any("response", response))
				} else if err != nil {
					h.logger.Info("retrying api after error", zap.String("err", err.Error()))
				} else {
					h.logger.Info("retrying api after error")
				}
				return true
			})
		}
	})

	// inject opentracing in the outgoing request
	tracer := opentracing.GlobalTracer()
	_ = tracer.Inject(sp.Context(), opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))

	// Set header
	if request.Header != nil {
		for name, headers := range request.Header {
			for _, value := range headers {
				r.SetHeader(name, value)
			}
		}
	}

	// Auto set application/json as default
	if _, ok := request.Header["Content-Type"]; !ok {
		if _, ok := request.Header["content-type"]; !ok {
			r.SetHeader("content-type", "application/json")
		}
	}

	// Set query param
	if request.QueryParam != nil {
		for name, values := range request.QueryParam {
			for _, value := range values {
				r.SetQueryParam(name, value)
			}
		}
	}

	// Set path param
	if request.PathParam != nil {
		for name, values := range request.PathParam {
			for _, value := range values {
				r.SetPathParam(name, value)
			}
		}
	}

	if b, ok := request.Body.([]byte); ok {
		r.SetBody(b)
	} else if request.BodyProvider != nil {
		if b, err := request.BodyProvider.Body(request.Body); err == nil {
			r.SetBody(b)
		} else {
			return nil, &command.GoxHttpError{
				Err:        err,
				StatusCode: http.StatusInternalServerError,
				Message:    "failed to read body using body provider",
				ErrorCode:  command.ErrorCodeFailedToBuildRequest,
			}
		}
	} else {
		if b, err := serialization.Stringify(request.Body); err == nil {
			r.SetBody(b)
		} else {
			return nil, &command.GoxHttpError{
				Err:        err,
				StatusCode: http.StatusInternalServerError,
				Message:    "failed to read body using Stringify",
				ErrorCode:  command.ErrorCodeFailedToBuildRequest,
			}
		}
	}

	return r, nil
}

func (h *HttpCommand) processResponse(request *command.GoxRequest, response *resty.Response) *command.GoxResponse {
	var processedResponse interface{}
	var err error

	if response.IsError() {

		if h.api.IsHttpCodeAcceptable(response.StatusCode()) {
			if request.ResponseBuilder != nil && response.Body() != nil {
				processedResponse, err = request.ResponseBuilder.Response(response.Body())
				if err != nil {
					return &command.GoxResponse{
						Body:       response.Body(),
						StatusCode: response.StatusCode(),
						Err: &command.GoxHttpError{
							Err:        errors.Wrap(err, "failed to create response using response builder"),
							StatusCode: response.StatusCode(),
							Message:    "failed to create response using response builder",
							ErrorCode:  "failed_to_build_response_using_response_builder",
							Body:       response.Body(),
						},
					}
				}
			}
		} else {
			return &command.GoxResponse{
				Body:       response.Body(),
				StatusCode: response.StatusCode(),
				Err: &command.GoxHttpError{
					Err:        errors.Wrap(err, "got response with server with error"),
					StatusCode: response.StatusCode(),
					Message:    "got response from server with error",
					ErrorCode:  "server_response_with_error",
					Body:       response.Body(),
				},
			}
		}

		return &command.GoxResponse{
			StatusCode: response.StatusCode(),
			Body:       response.Body(),
			Response:   processedResponse,
		}

	} else {

		if request.ResponseBuilder != nil && response.Body() != nil {
			processedResponse, err = request.ResponseBuilder.Response(response.Body())
			if err != nil {
				return &command.GoxResponse{
					Body:       response.Body(),
					StatusCode: response.StatusCode(),
					Err: &command.GoxHttpError{
						Err:        errors.Wrap(err, "failed to create response using response builder"),
						StatusCode: response.StatusCode(),
						Message:    "failed to create response using response builder",
						ErrorCode:  "failed_to_build_response_using_response_builder",
						Body:       response.Body(),
					},
				}
			}
		}

		return &command.GoxResponse{
			StatusCode: response.StatusCode(),
			Body:       response.Body(),
			Response:   processedResponse,
		}
	}
}

func (h *HttpCommand) handleError(err error) *command.GoxResponse {
	var responseObject *command.GoxResponse

	// Timeout errors are handled here
	switch e := err.(type) {
	case net.Error:
		if e.Timeout() {
			responseObject = &command.GoxResponse{
				StatusCode: http.StatusRequestTimeout,
				Err: &command.GoxHttpError{
					Err:        e,
					StatusCode: http.StatusRequestTimeout,
					Message:    "request timeout on client",
					ErrorCode:  "request_timeout_on_client",
				},
			}
		}
	}

	// Not a timeout error
	if responseObject == nil {
		responseObject = &command.GoxResponse{
			StatusCode: http.StatusBadRequest,
			Err: &command.GoxHttpError{
				Err:        err,
				StatusCode: http.StatusBadRequest,
				Message:    "request failed on client",
				ErrorCode:  "request_failed_on_client",
			},
		}
	}

	return responseObject
}

func NewHttpCommand(cf gox.CrossFunction, server *command.Server, api *command.Api) (command.Command, error) {
	c := &HttpCommand{
		CrossFunction:    cf,
		server:           server,
		api:              api,
		logger:           cf.Logger().Named("goxHttp").Named(api.Name),
		client:           resty.New(),
		setRetryFuncOnce: &sync.Once{},
	}
	c.client.SetAllowGetMethodPayload(true)
	c.client.SetTimeout(time.Duration(api.Timeout) * time.Millisecond)
	return c, nil
}
