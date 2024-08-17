package goxHttpApi

import (
	"bytes"
	"fmt"
	"github.com/devlibx/gox-base/v2"
	"github.com/gin-gonic/gin"
	"io"
	"log/slog"
	"strings"
	"time"
)

type RequestResponseLog struct {
	URL               string
	RequestHeaders    map[string]interface{}
	ResponseHeaders   map[string]interface{}
	RequestBodyBytes  []byte
	ResponseBodyBytes []byte
	TimeTakenMs       int64
	Status            int
}

func (r *RequestResponseLog) String() string {
	var logBuilder strings.Builder
	logBuilder.WriteString(fmt.Sprintf("URL:\n %s  Status=%d, TimeMs=%d\n", r.URL, r.Status, r.TimeTakenMs))
	logBuilder.WriteString(fmt.Sprintf("Request Headers:\n %v\n", r.RequestHeaders))
	logBuilder.WriteString(fmt.Sprintf("Request Body:\n %s\n", r.RequestBodyBytes))
	logBuilder.WriteString(fmt.Sprintf("Response Headers:\n %v\n", r.ResponseHeaders))
	logBuilder.WriteString(fmt.Sprintf("Response Body:\n %s\n", r.ResponseBodyBytes))
	return logBuilder.String()
}

// requestResponseBodyWriter is a wrapper around gin.ResponseWriter that captures the response body
type requestResponseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w requestResponseBodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// RequestResponseLoggerMiddleware is a middleware that logs the request and response
func RequestResponseLoggerMiddleware(logFunc func(*gin.Context, *RequestResponseLog)) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Request headers
		requestHeaders := map[string]interface{}{}
		for key, values := range c.Request.Header {
			for _, value := range values {
				requestHeaders[key] = value
			}
		}

		// Capture full body and restore the io.ReadCloser to its original state
		var requestBodyBytes []byte
		if c.Request.Body != nil {
			requestBodyBytes, _ = io.ReadAll(c.Request.Body)
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBodyBytes))

		// Capture the response body
		responseBodyWriter := &requestResponseBodyWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = responseBodyWriter

		// Process request
		c.Next()

		// Response headers
		responseHeaders := map[string]interface{}{}
		for key, values := range c.Writer.Header() {
			for _, value := range values {
				responseHeaders[key] = value
			}
		}

		// Make sure we have logFunc in the input
		if logFunc == nil {
			return
		}

		// Report log
		log := &RequestResponseLog{
			Status:           responseBodyWriter.Status(),
			URL:              c.Request.URL.String(),
			RequestHeaders:   requestHeaders,
			ResponseHeaders:  responseHeaders,
			RequestBodyBytes: requestBodyBytes,
		}
		if responseBodyWriter.body != nil {
			log.ResponseBodyBytes = responseBodyWriter.body.Bytes()
		}
		end := time.Now()
		log.TimeTakenMs = end.UnixMilli() - start.UnixMilli()
		logFunc(c, log)
	}
}

// RequestResponseSecurityConfig is the configuration for request response security
type RequestResponseSecurityConfig struct {
	EnableRequestLoggingToConsole bool     `json:"enable_request_logging_to_console" yaml:"enable_request_logging_to_console"`
	EnableRequestLogging          bool     `json:"enable_request_logging" yaml:"enable_request_logging"`
	IgnoreRequestHeaders          []string `json:"ignore_request_headers" yaml:"ignore_request_headers"`
	IgnoreResponseHeaders         []string `json:"ignore_response_headers" yaml:"ignore_response_headers"`
	IgnoreKeysInRequest           []string `json:"ignore_keys_in_request" yaml:"ignore_keys_in_request"`
	IgnoreKeysInResponse          []string `json:"ignore_keys_in_response" yaml:"ignore_keys_in_response"`
	MaskString                    string   `json:"mask_string" yaml:"mask_string"`
}

// RequestResponseSecurityConfigApplier is an interface to apply security configuration to request response logs
type RequestResponseSecurityConfigApplier interface {
	Process(log *RequestResponseLog) *RequestResponseLog
}

// requestResponseSecurityConfigApplier is an implementation of RequestResponseSecurityConfigApplier
type requestResponseSecurityConfigApplier struct {
	Config *RequestResponseSecurityConfig `json:"config"`

	requestKeysToMaks  map[string]struct{}
	responseKeysToMaks map[string]struct{}
}

// NewRequestResponseSecurityConfigApplier creates a new RequestResponseSecurityConfigApplier
func NewRequestResponseSecurityConfigApplier(config *RequestResponseSecurityConfig) RequestResponseSecurityConfigApplier {
	r := &requestResponseSecurityConfigApplier{
		Config: config,
	}

	if r.Config != nil && r.Config.IgnoreKeysInRequest != nil {
		r.requestKeysToMaks = make(map[string]struct{})
		for _, key := range r.Config.IgnoreKeysInRequest {
			r.requestKeysToMaks[key] = struct{}{}
		}
	}
	if r.Config != nil && r.Config.IgnoreKeysInResponse != nil {
		r.responseKeysToMaks = make(map[string]struct{})
		for _, key := range r.Config.IgnoreKeysInResponse {
			r.responseKeysToMaks[key] = struct{}{}
		}
	}

	if r.Config != nil && r.Config.MaskString == "" {
		r.Config.MaskString = "****"
	}

	return r
}

func (r *requestResponseSecurityConfigApplier) Process(log *RequestResponseLog) *RequestResponseLog {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("failed requestResponseSecurityConfigApplier - handled using recover")
		}
	}()

	if r.Config == nil {
		return log
	} else if !r.Config.EnableRequestLogging {
		return &RequestResponseLog{
			URL:               "",
			RequestHeaders:    map[string]interface{}{},
			ResponseHeaders:   map[string]interface{}{},
			RequestBodyBytes:  []byte("{}"),
			ResponseBodyBytes: []byte("{}"),
		}
	}

	if r.Config.IgnoreRequestHeaders != nil {
		for _, header := range r.Config.IgnoreRequestHeaders {
			if _, ok := log.RequestHeaders[header]; ok {
				log.RequestHeaders[header] = r.Config.MaskString
			}
		}
	}

	if r.Config.IgnoreResponseHeaders != nil {
		for _, header := range r.Config.IgnoreResponseHeaders {
			if _, ok := log.ResponseHeaders[header]; ok {
				log.ResponseHeaders[header] = r.Config.MaskString
			}
		}
	}

	// Mask keys in request
	if log.RequestBodyBytes != nil && len(log.RequestBodyBytes) > 0 {
		if req, err := gox.StringObjectMapFromString(string(log.RequestBodyBytes)); err == nil {
			maskMapValues(req, r.requestKeysToMaks, r.Config.MaskString)
			log.RequestBodyBytes = []byte(req.JsonStringOrEmptyJson())
		}
	}

	// Mask keys in response
	if log.ResponseBodyBytes != nil && len(log.ResponseBodyBytes) > 0 {
		if resp, err := gox.StringObjectMapFromString(string(log.ResponseBodyBytes)); err == nil {
			maskMapValues(resp, r.responseKeysToMaks, r.Config.MaskString)
			log.ResponseBodyBytes = []byte(resp.JsonStringOrEmptyJson())
		}
	}

	return log
}

func maskMapValues(data gox.StringObjectMap, keysToMask map[string]struct{}, maskString string) {
	for key, value := range data {
		if _, found := keysToMask[key]; found {
			data[key] = maskString
		} else {
			// If the value is a nested map, recursively mask its values
			if nestedMap, ok := value.(map[string]interface{}); ok {
				maskMapValues(nestedMap, keysToMask, maskString)
			}
			// If the value is a slice, check for nested maps in the slice
			if nestedSlice, ok := value.([]interface{}); ok {
				for _, item := range nestedSlice {
					if nestedMap, ok := item.(map[string]interface{}); ok {
						maskMapValues(nestedMap, keysToMask, maskString)
					}
				}
			}
		}
	}
}
