package goxHttpApi

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"strings"
)

type RequestResponseLog struct {
	URL               string
	RequestHeaders    map[string]interface{}
	ResponseHeaders   map[string]interface{}
	RequestBodyBytes  []byte
	ResponseBodyBytes []byte
}

func (r *RequestResponseLog) String() string {
	var logBuilder strings.Builder
	logBuilder.WriteString(fmt.Sprintf("URL:\n %s\n", r.URL))
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
func RequestResponseLoggerMiddleware(logFunc func(*RequestResponseLog)) gin.HandlerFunc {
	return func(c *gin.Context) {
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
			URL:              c.Request.URL.String(),
			RequestHeaders:   requestHeaders,
			ResponseHeaders:  responseHeaders,
			RequestBodyBytes: requestBodyBytes,
		}
		if responseBodyWriter.body != nil {
			log.ResponseBodyBytes = responseBodyWriter.body.Bytes()
		}
		logFunc(log)
	}
}
