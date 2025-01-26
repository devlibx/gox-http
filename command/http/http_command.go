package httpCommand

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/devlibx/gox-base/v2"
	"github.com/devlibx/gox-base/v2/errors"
	"github.com/devlibx/gox-base/v2/serialization"
	"github.com/devlibx/gox-http/v4/command"
	"github.com/devlibx/gox-http/v4/interceptor"
	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var EnableGoxHttpMetricLogging = false
var EnableTimeTakenByHttpCall = false
var EnableRequestResponseBodyLogging = false
var EnableRestyDebug = false

// StartSpanFromContext is added for someone to override the implementation
type StartSpanFromContext func(ctx context.Context, operationName string, opts ...opentracing.StartSpanOption) (opentracing.Span, context.Context)

// DefaultStartSpanFromContextFunc provides a default implementation for StartSpanFromContext function
var DefaultStartSpanFromContextFunc StartSpanFromContext = func(ctx context.Context, operationName string, opts ...opentracing.StartSpanOption) (opentracing.Span, context.Context) {
	return opentracing.StartSpanFromContext(ctx, operationName, opts...)
}

var DefaultHttpTrackingFunc = func() {}

var ContextBasedHeaderPrefix = "__request_scope_headers__"

type HttpCommand struct {
	gox.CrossFunction
	server           *command.Server
	api              *command.Api
	logger           *zap.Logger
	debugLogger      *zap.SugaredLogger
	client           *resty.Client
	setRetryFuncOnce *sync.Once

	deepCopyOfApi *command.Api
}

// GetRestyClient method will return underlying resty client if it uses it
// Returns:
// - resty client if this command is implemented using resty client under the hood
// - bool - true if resty client is returned otherwise false
func (h *HttpCommand) GetRestyClient() (*resty.Client, bool) {
	return h.client, true
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

	// Run a registered pre-request interceptor
	if EnablePreRequestInterceptor {
		if stop, resp, err := h.interceptPreRequestInterceptor(ctx, request); stop {
			return resp, err
		}
	}

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

	var response *resty.Response

	// Build request with all parameters
	r, err := h.buildRequest(ctxWithSpan, request, sp)
	if err != nil {
		h.debugLogger.Debug("got request to execute (err)", zap.Stringer("request", request))
		return nil, err
	}

	// Create the url to call
	finalUrlToRequest := h.api.GetPath(h.server)
	if !EnableRequestResponseBodyLogging {
		h.debugLogger.Debug("got request to execute", zap.Stringer("request", request), zap.String("url", finalUrlToRequest))
	}

	// Track http connection if enabled
	ht := HttpCallTracking{}
	ht.trackHttp(request, r, h.api, h.server)

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
	case "PATCH":
		response, err = r.Patch(finalUrlToRequest)
	}
	end := time.Now()

	urlToPrint := finalUrlToRequest
	if EnableTimeTakenByHttpCall {
		if response != nil && response.Request != nil && response.Request.URL != "" {
			urlToPrint = response.Request.URL
		}
		h.logger.Info("Time taken: ", zap.Int64("time_taken", end.UnixMilli()-start.UnixMilli()), zap.Int64("start", start.UnixMilli()), zap.Int64("end", end.UnixMilli()), zap.String("url", urlToPrint))
	}

	if EnableRequestResponseBodyLogging {
		if response.Body() != nil && len(response.Body()) > 0 {
			h.debugLogger.Debug("request/response of http call", zap.String("url", finalUrlToRequest), zap.Stringer("request", request), zap.String("response", string(response.Body())), zap.Int("response_code", response.StatusCode()))
		} else {
			h.debugLogger.Debug("request/response of http call", zap.String("url", finalUrlToRequest), zap.Stringer("request", request), zap.Int("response_code", response.StatusCode()))
		}
	}

	// Send tracking event to be processed
	defer func() {
		h.publishTracking(request, r, finalUrlToRequest, ht)
	}()

	if err != nil {
		responseObject := h.handleError(err)
		return responseObject, responseObject.Err
	} else {
		responseObject := h.processResponse(request, response)
		return responseObject, responseObject.Err
	}
}

func (h *HttpCommand) publishTracking(request *command.GoxRequest, r *resty.Request, fullPath string, tracingEvent HttpCallTracking) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("publishTracking Recovered from panic: %v", r)
		}
	}()

	if HttpTrackingFuncSingleton != nil {
		HttpTrackingFuncSingleton(request, r, fullPath, tracingEvent)
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

	// Add MDC which
	if h.server.Properties != nil {
		if _mdc, ok := h.server.Properties["mdc"]; ok {
			if mdc, ok := _mdc.(string); ok {
				for _, k := range strings.Split(mdc, ",") {
					if v := ctx.Value(k); v != nil {
						if s, ok := v.(string); ok {
							r.SetHeader(k, s)
						}
					}
				}
			}
		}
	}

	// Set headers from service
	if h.server.Headers != nil {
		for name, value := range h.server.Headers {
			if name == "__UNIQUE_UUID__" {
				r.SetHeader(name, uuid.New().String())
			} else {
				if s, ok := value.(string); ok {
					r.SetHeader(name, s)
				} else {
					r.SetHeader(name, serialization.StringifyOrEmptyJsonOnError(value))
				}
			}
		}
	}

	// Set default headers
	if h.api.Headers != nil {
		for name, value := range h.api.Headers {
			if name == "__UNIQUE_UUID__" {
				r.SetHeader(name, uuid.New().String())
			} else {
				r.SetHeader(name, value)
			}
		}
	}

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

	// Add request scope header
	if ctx.Value(ContextBasedHeaderPrefix) != nil {
		if requestScopeHeaders, ok := ctx.Value(ContextBasedHeaderPrefix).(map[string]interface{}); ok {
			for k, v := range requestScopeHeaders {
				r.SetHeader(k, fmt.Sprintf("%v", v))
			}
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
	} else if request.Body != nil {
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

	return h.intercept(ctx, r)
}

func (h *HttpCommand) intercept(ctx context.Context, r *resty.Request) (*resty.Request, error) {

	// Get the valid config from server and api
	var interceptorConfig *interceptor.Config
	if h.server.InterceptorConfig != nil && !h.server.InterceptorConfig.Disabled {
		interceptorConfig = h.server.InterceptorConfig
	} else if h.api.InterceptorConfig != nil && !h.api.InterceptorConfig.Disabled {
		interceptorConfig = h.api.InterceptorConfig
	} else {
		return r, nil
	}

	// Before we move forward, lets intercept this request and enrich it if required
	in := interceptor.NewInterceptor(interceptorConfig)
	if _, enabled := in.Info(); !enabled {
		return r, nil
	}

	// Name to log error
	name, _ := in.Info()

	// Intercept body and update if required
	if requestModified, modifiedRequest, err := in.Intercept(ctx, r); err != nil {
		return nil, errors.Wrap(err, "failed to intercept request body using interceptor: name=%s", name)
	} else if requestModified {
		return modifiedRequest.(*resty.Request), nil
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
	var e net.Error
	switch {
	case errors.As(err, &e):
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

	// We need to build a client and also consider if we need to use proxy or not
	var client *resty.Client
	if server.ProxyUrl == "" {
		client = resty.New()
	} else {
		if proxyURL, err := url.Parse(server.ProxyUrl); err != nil {
			return nil, errors.Wrap(err, "failed to parse proxy url: url=%s", server.ProxyUrl)
		} else {
			client = resty.New()
			client.SetTransport(&http.Transport{Proxy: http.ProxyURL(proxyURL)})
			if server.SkipCertVerify == "true" {
				client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
			}
		}
	}

	c := &HttpCommand{
		CrossFunction:    cf,
		server:           server,
		api:              api,
		logger:           cf.Logger().Named("goxHttp").Named(api.Name),
		client:           client,
		setRetryFuncOnce: &sync.Once{},
	}
	c.debugLogger = c.logger.Sugar()
	c.client.SetAllowGetMethodPayload(true)
	c.client.SetTimeout(time.Duration(api.Timeout) * time.Millisecond)

	// If Resty Debug is enabled then we will dump request response
	if EnableRestyDebug || api.EnableRequestResponseLogging {
		c.client.SetDebug(true)
	}

	return c, nil
}
