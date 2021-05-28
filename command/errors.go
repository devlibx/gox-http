package command

import (
	"fmt"
	"github.com/devlibx/gox-base/util"
	"net/http"
)

const ErrorCodeFailedToBuildRequest = "failed_to_build_request"
const ErrorCodeFailedToRequestServer = "failed_to_request_server"

type GoxHttpError struct {
	Err        error
	StatusCode int
	Message    string
	ErrorCode  string
	Body       []byte
}

// Build string representation
func (e *GoxHttpError) Error() string {
	if !util.IsStringEmpty(e.Message) && !util.IsStringEmpty(e.ErrorCode) {
		return fmt.Sprintf("statusCode=%d, message=%s, errorCode=%s, err=%v", e.StatusCode, e.Message, e.ErrorCode, e.Err)
	} else if !util.IsStringEmpty(e.Message) {
		return fmt.Sprintf("statusCode=%d, message=%s, err=%v", e.StatusCode, e.Message, e.Err)
	} else if !util.IsStringEmpty(e.Message) {
		return fmt.Sprintf("statusCode=%d, errorCode=%s, err=%v", e.StatusCode, e.ErrorCode, e.Err)
	} else {
		return fmt.Sprintf("statusCode=%d, err=%v", e.StatusCode, e.Err)
	}
}

// Build string representation
func (e *GoxHttpError) Unwrap() error {
	return e.Err
}

func (e *GoxHttpError) Is2xx() bool {
	return e.StatusCode >= http.StatusOK && e.StatusCode <= http.StatusIMUsed
}

func (e *GoxHttpError) Is3xx() bool {
	return e.StatusCode >= http.StatusBadRequest && e.StatusCode <= http.StatusUnavailableForLegalReasons
}

func (e *GoxHttpError) Is4xx() bool {
	return e.StatusCode >= http.StatusBadRequest && e.StatusCode <= http.StatusUnavailableForLegalReasons
}

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