package goxHttpApi

import (
	"context"
	"github.com/devlibx/gox-base/errors"
	"github.com/devlibx/gox-base/serialization"
	"github.com/devlibx/gox-http/v2/command"
	"net/http"
)

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

// ExtractError is a helper function to extract error from error object
//
// first return value is the error object
// second [errorResponsePayloadExists[bool]] is true if error response payload exists. There are cases where you
// may have error by error body also exist, in such case this will be true
// third [errorExists[bool]] is true if error exists
//
// NOTE 0 if oyu got the error from ExecuteHttp then you can ignore checking OK
func ExtractError[ErrorResp any](err error) (errorResp *GoxError[*ErrorResp], errorResponsePayloadExists bool, ok bool) {
	var e *GoxError[*ErrorResp]
	if errors.As(err, &e) {
		if e.Response == nil {
			return e, false, true
		} else {
			return e, true, true
		}
	}
	return nil, false, false
}

// ExecuteHttp is a helper function to execute http request and parse response to success or error object
func ExecuteHttp[SuccessResp any, ErrorResp any](
	cxt context.Context,
	goxHttpCtx GoxHttpContext,
	request *command.GoxRequest,
) (*GoxSuccessResponse[SuccessResp], error) {

	resp, err := goxHttpCtx.Execute(cxt, request)
	if err == nil {

		// If status is StatusNoContent then we will do special handling
		if resp.StatusCode == http.StatusNoContent {
			return &GoxSuccessResponse[SuccessResp]{
				StatusCode: http.StatusNoContent,
			}, nil
		}

		// In other cases we will parse the response
		var successResp SuccessResp
		if serializationErr := serialization.JsonBytesToObject(resp.Body, &successResp); serializationErr == nil {
			return &GoxSuccessResponse[SuccessResp]{
				Body:       resp.Body,
				StatusCode: resp.StatusCode,
				Response:   successResp,
			}, nil
		} else {
			return nil, &GoxError[*ErrorResp]{
				Body:       resp.Body,
				StatusCode: resp.StatusCode,
				Err:        errors.Wrap(err, "http request passed but failed to parse response into response object"),
			}
		}
	}

	var goxError *command.GoxHttpError
	if errors.As(err, &goxError) {
		var errorResp *ErrorResp
		if serializationErr := serialization.JsonBytesToObject(resp.Body, &errorResp); serializationErr == nil {
			return nil, &GoxError[*ErrorResp]{
				Response:   errorResp,
				Body:       resp.Body,
				StatusCode: resp.StatusCode,
				Err:        err,
			}
		} else {
			return nil, &GoxError[*ErrorResp]{
				Body:       resp.Body,
				StatusCode: resp.StatusCode,
				Err:        errors.Wrap(err, "http request got error with response but failed to parse response into response object"),
			}
		}
	}

	return nil, &GoxError[*ErrorResp]{
		StatusCode: http.StatusInternalServerError,
		Err:        errors.Wrap(err, "http request failed"),
	}
}

// ExecuteHttpListResponse is a helper function to execute http request and parse response to success or error object
func ExecuteHttpListResponse[SuccessResp any, ErrorResp any](
	cxt context.Context,
	goxHttpCtx GoxHttpContext,
	request *command.GoxRequest,
) (*GoxSuccessListResponse[SuccessResp], error) {

	resp, err := goxHttpCtx.Execute(cxt, request)
	if err == nil {

		// If status is StatusNoContent then we will do special handling
		if resp.StatusCode == http.StatusNoContent {
			return &GoxSuccessListResponse[SuccessResp]{
				StatusCode: http.StatusNoContent,
			}, nil
		}

		// In other cases we will parse the response
		var successResp []SuccessResp
		if serializationErr := serialization.JsonBytesToObject(resp.Body, &successResp); serializationErr == nil {
			return &GoxSuccessListResponse[SuccessResp]{
				Body:       resp.Body,
				StatusCode: resp.StatusCode,
				Response:   successResp,
			}, nil
		} else {
			return nil, &GoxError[*ErrorResp]{
				Body:       resp.Body,
				StatusCode: resp.StatusCode,
				Err:        errors.Wrap(err, "http request passed but failed to parse response into response object"),
			}
		}
	}

	var goxError *command.GoxHttpError
	if errors.As(err, &goxError) {
		var errorResp *ErrorResp
		if serializationErr := serialization.JsonBytesToObject(resp.Body, &errorResp); serializationErr == nil {
			return nil, &GoxError[*ErrorResp]{
				Response:   errorResp,
				Body:       resp.Body,
				StatusCode: resp.StatusCode,
				Err:        err,
			}
		} else {
			return nil, &GoxError[*ErrorResp]{
				Body:       resp.Body,
				StatusCode: resp.StatusCode,
				Err:        errors.Wrap(err, "http request got error with response but failed to parse response into response object"),
			}
		}
	}

	return nil, &GoxError[*ErrorResp]{
		StatusCode: http.StatusInternalServerError,
		Err:        errors.Wrap(err, "http request failed"),
	}
}
