package command

import (
	"fmt"
	"github.com/devlibx/gox-base/util"
	"net/http"
)

const ErrorCodeFailedToBuildRequest = "failed_to_build_request"
const ErrorCodeFailedToRequestServer = "failed_to_request_server"

// Gox Http Module error
// Err 			- underlying error thrown by http or lib
// StatusCode 	- http status code
// Message 		- human readable code for debugging
// ErrorCode	- pre-defined error codes
// Body			- data from http response
//
//	This may be nil if we got local errors e.g. hystrix timeout, or some other errors
type GoxHttpError struct {
	Err        error
	StatusCode int
	Message    string
	ErrorCode  string
	Body       []byte
}

// Build string representation
func (e *GoxHttpError) Error() string {
	body := "<no body from server>"
	if e.Body != nil {
		body = string(e.Body)
	}
	if !util.IsStringEmpty(e.Message) && !util.IsStringEmpty(e.ErrorCode) {
		return fmt.Sprintf("statusCode=%d, message=%s, body=%s, errorCode=%s, err=%v", e.StatusCode, e.Message, body, e.ErrorCode, e.Err)
	} else if !util.IsStringEmpty(e.Message) {
		return fmt.Sprintf("statusCode=%d, message=%s, body=%s, err=%v", e.StatusCode, e.Message, body, e.Err)
	} else if !util.IsStringEmpty(e.Message) {
		return fmt.Sprintf("statusCode=%d, body=%s, errorCode=%s, err=%v", e.StatusCode, body, e.ErrorCode, e.Err)
	} else {
		return fmt.Sprintf("statusCode=%d, body=%s, err=%v", e.StatusCode, body, e.Err)
	}
}

// Build string representation
func (e *GoxHttpError) Unwrap() error {
	return e.Err
}

func (e *GoxHttpError) Is2xx() bool {
	return e.StatusCode >= http.StatusOK && e.StatusCode <= http.StatusIMUsed
}

// Is this 3xx error
func (e *GoxHttpError) Is3xx() bool {
	return e.StatusCode >= http.StatusBadRequest && e.StatusCode <= http.StatusUnavailableForLegalReasons
}

// Is this 4xx error
func (e *GoxHttpError) Is4xx() bool {
	return e.StatusCode >= http.StatusBadRequest && e.StatusCode <= http.StatusUnavailableForLegalReasons
}

// Is this 5xx error
func (e *GoxHttpError) Is5xx() bool {
	return e.StatusCode >= http.StatusInternalServerError && e.StatusCode <= http.StatusNetworkAuthenticationRequired
}

func (e *GoxHttpError) IsInternalServerError() bool {
	return e.StatusCode == http.StatusInternalServerError
}

func (e *GoxHttpError) IsBadGateway() bool {
	return e.StatusCode == http.StatusBadGateway
}

func (e *GoxHttpError) IsServiceUnavailable() bool {
	return e.StatusCode == http.StatusServiceUnavailable
}

func (e *GoxHttpError) IsGatewayTimeout() bool {
	return e.StatusCode == http.StatusGatewayTimeout
}

func (e *GoxHttpError) IsBadRequest() bool {
	return e.StatusCode == http.StatusBadRequest
}

func (e *GoxHttpError) IsUnauthorized() bool {
	return e.StatusCode == http.StatusUnauthorized || e.StatusCode == http.StatusForbidden
}

func (e *GoxHttpError) IsNotFound() bool {
	return e.StatusCode == http.StatusNotFound
}

func (e *GoxHttpError) IsMethodNotAllowed() bool {
	return e.StatusCode == http.StatusMethodNotAllowed
}

func (e *GoxHttpError) IsNotAcceptable() bool {
	return e.StatusCode == http.StatusNotAcceptable
}

func (e *GoxHttpError) IsRequestTimeout() bool {
	return e.StatusCode == http.StatusRequestTimeout
}

func (e *GoxHttpError) IsConflict() bool {
	return e.StatusCode == http.StatusConflict
}

// Indicates that this error was caused because hystrix circuit is open due to too many errors
func (e *GoxHttpError) IsHystrixCircuitOpenError() bool {
	return e.ErrorCode == "hystrix_circuit_open"
}

// Indicates that this error was caused because the command took longer then hystrix configured time
func (e *GoxHttpError) IsHystrixTimeoutError() bool {
	return e.ErrorCode == "hystrix_timeout"
}

// Indicates that this error was caused because too many requests are submitted
func (e *GoxHttpError) IsHystrixRejectedError() bool {
	return e.ErrorCode == "hystrix_rejected"
}

// Indicates that this error was caused due to hystrix issue (timeout/circuit open/rejected)
func (e *GoxHttpError) IsHystrixError() bool {
	return e.IsHystrixTimeoutError() || e.IsHystrixCircuitOpenError() || e.IsHystrixRejectedError()
}
