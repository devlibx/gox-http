package goxHttpApi

import (
	"context"
	"github.com/devlibx/gox-base/v2/errors"
	"github.com/devlibx/gox-base/v2/serialization"
	"github.com/devlibx/gox-http/v4/command"
	"github.com/opentracing/opentracing-go"
	"log/slog"
	"net/http"
)

// GoxHttpRequestResponseLoggingEnabled is a global flag to enable logging of request response
var GoxHttpRequestResponseLoggingEnabled = slog.LevelDebug

// GoxSuccessResponse is the typed response after successful http call and parsing response to success object
type GoxSuccessResponse[SuccessResp any] struct {
	Body       []byte
	Response   SuccessResp
	StatusCode int
}

// GoxSuccessListResponse is the typed response after successful http call and parsing response to success object
type GoxSuccessListResponse[SuccessResp any] struct {
	Body       []byte
	Response   []SuccessResp
	StatusCode int
}

// GoxError is the typed response after successful http call and parsing response to success object
type GoxError[ErrorResp any] struct {
	Body       []byte
	Response   ErrorResp
	StatusCode int
	Err        error
}

// ExecuteHttp is a helper function to execute http request and parse response to success or error object
func ExecuteHttp[SuccessResp any, ErrorResp any](
	ctx context.Context,
	goxHttpCtx GoxHttpContext,
	request *command.GoxRequest,
) (*GoxSuccessResponse[SuccessResp], error) {

	// Execute HTTP
	resp, _, err := internalExecuteHttp[SuccessResp, ErrorResp](ctx, goxHttpCtx, request, false)

	// If log is enabled then dump based on log level
	if GoxHttpRequestResponseLoggingEnabled >= slog.LevelDebug {
		logRequestResponse[SuccessResp, ErrorResp](request, resp, GoxHttpRequestResponseLoggingEnabled)
	}

	return resp, err
}

// ExecuteHttpListResponse is a helper function to execute http request and parse list based response to success or error object
func ExecuteHttpListResponse[SuccessResp any, ErrorResp any](
	ctx context.Context,
	goxHttpCtx GoxHttpContext,
	request *command.GoxRequest,
) (*GoxSuccessListResponse[SuccessResp], error) {
	_, resp, err := internalExecuteHttp[SuccessResp, ErrorResp](ctx, goxHttpCtx, request, true)
	return resp, err
}

// internalExecuteHttp is a helper function to execute http request and parse response to success or error object
func internalExecuteHttp[SuccessResp any, ErrorResp any](
	ctx context.Context,
	goxHttpCtx GoxHttpContext,
	request *command.GoxRequest,
	isList bool,
) (*GoxSuccessResponse[SuccessResp], *GoxSuccessListResponse[SuccessResp], error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "goxHttp-"+request.Api)
	defer span.Finish()

	// Execute request and process response
	resp, err := goxHttpCtx.Execute(ctx, request)
	if err == nil {
		if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
			return processSuccess[SuccessResp, ErrorResp](resp, isList, err)
		} else {
			logSpanOnError(span, err, request)
			return processError[SuccessResp, ErrorResp](err, resp)
		}
	} else {
		logSpanOnError(span, err, request)
		return processError[SuccessResp, ErrorResp](err, resp)
	}
}

func processSuccess[SuccessResp any, ErrorResp any](resp *command.GoxResponse, isList bool, err error) (*GoxSuccessResponse[SuccessResp], *GoxSuccessListResponse[SuccessResp], error) {

	// If status is StatusNoContent then we will do special handling
	if resp.StatusCode == http.StatusNoContent {
		return &GoxSuccessResponse[SuccessResp]{StatusCode: http.StatusNoContent},
			&GoxSuccessListResponse[SuccessResp]{StatusCode: http.StatusNoContent},
			nil
	} else if resp.StatusCode == http.StatusOK && (resp.Body == nil || len(resp.Body) == 0) {
		// There are cases where we get http status = 200 but no response body
		return &GoxSuccessResponse[SuccessResp]{StatusCode: http.StatusNoContent},
			&GoxSuccessListResponse[SuccessResp]{StatusCode: http.StatusNoContent},
			nil
	}

	// If this is other status then based on the list/not-list process the response
	if isList {
		var successResp []SuccessResp
		if serializationErr := serialization.JsonBytesToObject(resp.Body, &successResp); serializationErr == nil {
			return nil, &GoxSuccessListResponse[SuccessResp]{
				Body:       resp.Body,
				StatusCode: resp.StatusCode,
				Response:   successResp,
			}, nil
		} else {
			err = serializationErr
		}
	} else {
		var successResp SuccessResp
		if serializationErr := serialization.JsonBytesToObject(resp.Body, &successResp); serializationErr == nil {
			return &GoxSuccessResponse[SuccessResp]{
				Body:       resp.Body,
				StatusCode: resp.StatusCode,
				Response:   successResp,
			}, nil, nil
		} else {
			err = serializationErr
		}
	}

	// We must have got error in parsing payload
	return nil, nil, &GoxError[ErrorResp]{
		Body:       resp.Body,
		StatusCode: resp.StatusCode,
		Err:        errors.Wrap(err, "http request passed but failed to parse response into response object"),
	}
}

func processError[SuccessResp any, ErrorResp any](err error, resp *command.GoxResponse) (*GoxSuccessResponse[SuccessResp], *GoxSuccessListResponse[SuccessResp], error) {
	var goxError *command.GoxHttpError
	if errors.As(err, &goxError) || (resp != nil && resp.Body != nil && len(resp.Body) > 0) {
		var errorResp ErrorResp
		if serializationErr := serialization.JsonBytesToObject(resp.Body, &errorResp); serializationErr == nil {
			return nil, nil, &GoxError[ErrorResp]{
				Response:   errorResp,
				Body:       resp.Body,
				StatusCode: resp.StatusCode,
				Err:        err,
			}
		} else {
			return nil, nil, &GoxError[ErrorResp]{
				Body:       resp.Body,
				StatusCode: resp.StatusCode,
				Err:        errors.Wrap(err, "http request got error with response but failed to parse response into response object"),
			}
		}
	}

	return nil, nil, &GoxError[ErrorResp]{
		StatusCode: http.StatusInternalServerError,
		Err:        errors.Wrap(err, "http request failed"),
	}
}

func logSpanOnError(span opentracing.Span, err error, request *command.GoxRequest) {
	// Make sure to log error in span
	if span != nil {
		span.SetTag("error", true)
		if err != nil {
			span.SetTag("message", err.Error())
		}
		span.SetTag("section", request.Api+" failed")
	}
}

func logRequestResponse[SuccessResp any, ErrorResp any](request *command.GoxRequest, response *GoxSuccessResponse[SuccessResp], level slog.Level) {
	apiLog := slog.String("api", request.Api)
	requestLog := slog.Any("request", request)
	responseLog := slog.String("response", "null")
	responseStrLog := slog.String("response_str", "null")
	statusLog := slog.Int("status", 0)
	if response != nil {
		if response.Body != nil {
			responseLog = slog.Any("response", response.Response)
			responseStrLog = slog.Any("response_str", string(response.Body))
		}
		statusLog = slog.Int("status", response.StatusCode)
	}
	if level == slog.LevelDebug {
		slog.Debug("GoxHttp Logging", apiLog, requestLog, responseLog, responseStrLog, statusLog)
	} else if level == slog.LevelInfo {
		slog.Info("GoxHttp Logging", apiLog, requestLog, responseLog, responseStrLog, statusLog)
	}
}

// ExtractError is a helper function to extract error from error object
//
// first return value is the error object
// second [errorResponsePayloadExists[bool]] is true if error response payload exists. There are cases where you
// may have error by error body also exist, in such case this will be true
// third [errorExists[bool]] is true if error exists
//
// NOTE 0 if oyu got the error from ExecuteHttp then you can ignore checking OK
func ExtractError[ErrorResp any](err error) (errorResp *GoxError[ErrorResp], errorResponsePayloadExists bool, ok bool) {
	var e *GoxError[ErrorResp]
	if errors.As(err, &e) {
		if e.Body == nil || len(e.Body) == 0 {
			return e, false, true
		} else {
			return e, true, true
		}
	}
	return nil, false, false
}

// Error is the error message
func (e *GoxError[ErrorResp]) Error() string {
	if e.Err == nil {
		return "http error response"
	}
	return e.Err.Error()
}

// Unwrap is the error message
func (e *GoxError[ErrorResp]) Unwrap() error {
	return e.Err
}
