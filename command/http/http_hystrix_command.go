package httpCommand

import (
	"context"
	"errors"
	errors2 "errors"
	"fmt"
	"github.com/afex/hystrix-go/hystrix"
	"github.com/devlibx/gox-base/v2"
	goxError "github.com/devlibx/gox-base/v2/errors"
	"github.com/devlibx/gox-http/v4/command"
	"github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
	"net/http"
)

var HystrixConfigMap = gox.StringObjectMap{}

type HttpHystrixCommand struct {
	gox.CrossFunction
	logger             *zap.Logger
	command            command.Command
	hystrixCommandName string
	api                *command.Api

	serverName string
	apiName    string
}

func (h *HttpHystrixCommand) UpdateCommand(command command.Command) {
	h.command = command
}

type result struct {
	response *command.GoxResponse
	err      error
}

func (h *HttpHystrixCommand) Execute(ctx context.Context, request *command.GoxRequest) (*command.GoxResponse, error) {
	r := &result{}
	if err := hystrix.Do(h.hystrixCommandName, func() error {
		r.response, r.err = h.command.Execute(ctx, request)
		h.logHystrixError(ctx, request, r.err)
		return r.err
	}, nil); err != nil {
		h.logHystrixError(ctx, request, err)
		return r.response, h.errorCreator(err)
	} else {
		h.logHystrixError(ctx, request, r.err)
		return r.response, r.err
	}
}

// If this is a hystrix error then log it
func (h *HttpHystrixCommand) logHystrixError(ctx context.Context, request *command.GoxRequest, err error) {
	var e hystrix.CircuitError
	if errors.As(err, &e) {
		span, _ := opentracing.StartSpanFromContext(ctx, h.hystrixCommandName+"_hystrix_error")
		defer span.Finish()
		span.SetTag("error", err)
		span.SetTag("error_type", e.Error())

		if EnableGoxHttpMetricLogging {
			if errors2.Is(e, hystrix.ErrMaxConcurrency) {
				h.Metric().Tagged(map[string]string{"server": h.serverName, "api": h.apiName, "status": fmt.Sprintf("%d", 500), "error": "hystrix_error__max_concurrency"}).Counter("gox_http_call").Inc(1)
			} else if errors2.Is(e, hystrix.ErrCircuitOpen) {
				h.Metric().Tagged(map[string]string{"server": h.serverName, "api": h.apiName, "status": fmt.Sprintf("%d", 500), "error": "hystrix_error__circuit_open"}).Counter("gox_http_call").Inc(1)
			} else if errors2.Is(e, hystrix.ErrTimeout) {
				h.Metric().Tagged(map[string]string{"server": h.serverName, "api": h.apiName, "status": fmt.Sprintf("%d", 500), "error": "hystrix_error__timeout"}).Counter("gox_http_call").Inc(1)
			} else {
				h.Metric().Tagged(map[string]string{"server": h.serverName, "api": h.apiName, "status": fmt.Sprintf("%d", 500), "error": "hystrix_error"}).Counter("gox_http_call").Inc(1)
			}
		}
	}
}

func (h *HttpHystrixCommand) ExecuteAsync(ctx context.Context, request *command.GoxRequest) chan *command.GoxResponse {
	return nil
}

func (h *HttpHystrixCommand) errorCreator(err error) error {
	var e *command.GoxHttpError
	if errors.As(err, &e) {
		return e
	}

	var e1 hystrix.CircuitError
	switch {
	case errors.As(err, &e1):
		if errors.Is(e1, hystrix.ErrCircuitOpen) {
			return &command.GoxHttpError{
				Err:        e1,
				StatusCode: http.StatusBadRequest,
				Message:    "hystrix circuit open",
				ErrorCode:  "hystrix_circuit_open",
				Body:       nil,
			}
		} else if errors.Is(e1, hystrix.ErrMaxConcurrency) {
			return &command.GoxHttpError{
				Err:        e1,
				StatusCode: http.StatusBadRequest,
				Message:    "hystrix rejected",
				ErrorCode:  "hystrix_rejected",
				Body:       nil,
			}
		} else if errors.Is(e1, hystrix.ErrTimeout) {
			return &command.GoxHttpError{
				Err:        e1,
				StatusCode: http.StatusBadRequest,
				Message:    "hystrix timeout",
				ErrorCode:  "hystrix_timeout",
				Body:       nil,
			}
		} else {
			return &command.GoxHttpError{
				Err:        e1,
				StatusCode: http.StatusBadRequest,
				Message:    "hystrix unknown ",
				ErrorCode:  "hystrix_unknown_error",
				Body:       nil,
			}
		}
	}

	return &command.GoxHttpError{
		Err:        err,
		StatusCode: http.StatusBadRequest,
		Message:    "unknown error",
		ErrorCode:  "unknown_error",
		Body:       nil,
	}
}

func NewHttpHystrixCommand(cf gox.CrossFunction, server *command.Server, api *command.Api) (command.Command, error) {

	hc, err := NewHttpCommand(cf, server, api)
	if err != nil {
		return nil, goxError.Wrap(err, "failed to crate http command for %s", api.Name)
	}

	// name to register hystrix
	commandName := api.Name

	c := &HttpHystrixCommand{
		CrossFunction:      cf,
		logger:             cf.Logger().Named("goxHttp").Named(api.Name),
		command:            hc,
		hystrixCommandName: commandName,
		api:                api,
		serverName:         server.Name,
		apiName:            api.Name,
	}

	// Set timeout + 10% delta
	timeout := api.Timeout

	// Add extra time to handle retry counts
	if api.RetryCount > 0 {
		timeout = timeout + (timeout * api.RetryCount) + api.InitialRetryWaitTimeMs
	}

	if timeout/10 <= 0 {
		timeout += 2
	} else {
		timeout += timeout / 10
	}

	// Inject setting - mostly used in testing
	config := HystrixConfigMap.StringObjectMapOrEmpty(api.Name)
	if config.IntOrZero("timeout") > 0 {
		timeout = config.IntOrZero("timeout")
	}

	hystrix.ConfigureCommand(commandName, hystrix.CommandConfig{
		Timeout:               timeout,
		MaxConcurrentRequests: api.Concurrency,
		ErrorPercentThreshold: 25,
	})

	return c, nil
}
