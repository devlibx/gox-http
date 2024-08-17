package goxHttpApi

import (
	"context"
	"fmt"
	"github.com/devlibx/gox-base/v2"
	"github.com/devlibx/gox-base/v2/serialization"
	"github.com/devlibx/gox-base/v2/test"
	"github.com/devlibx/gox-http/v2/command"
	"github.com/devlibx/gox-http/v2/testhelper"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func Test_Post_Success(t *testing.T) {
	cf, _ := test.MockCf(t)

	// Setup sample response
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if body, err := ioutil.ReadAll(r.Body); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			if _body, err := gox.StringObjectMapFromString(string(body)); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				data := gox.StringObjectMap{"status": "ok"}
				for k, v := range _body {
					data[k] = v
				}
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprintln(w, serialization.StringifySuppressError(data, "{}"))
			}
		}
	}))
	defer ts.Close()

	// Read config and put the port to call
	config := command.Config{}
	err := serialization.ReadYamlFromString(testhelper.TestConfigWithRealServer, &config)
	assert.NoError(t, err)
	config.Servers["testServer"].Port, err = strconv.Atoi(strings.ReplaceAll(ts.URL, "http://127.0.0.1:", ""))
	assert.NoError(t, err)
	config.Apis["delay_timeout_10_POST"].DisableHystrix = true

	// Setup goHttp context
	goxHttpCtx, err := NewGoxHttpContext(cf, &config)
	assert.NoError(t, err)

	// Test 1 - Call http to get data
	ctx, ctxC := context.WithTimeout(context.Background(), 2*time.Second)
	defer ctxC()

	request := command.NewGoxRequestBuilder("delay_timeout_10_POST").
		WithContentTypeJson().
		WithPathParam("id", 1).
		WithBody(gox.StringObjectMap{"key": "value"}).
		WithResponseBuilder(command.NewJsonToObjectResponseBuilder(&gox.StringObjectMap{})).
		Build()
	response, err := goxHttpCtx.Execute(ctx, request)
	assert.NoError(t, err)
	assert.Equal(t, "ok", response.AsStringObjectMapOrEmpty().StringOrEmpty("status"))
	assert.Equal(t, "value", response.AsStringObjectMapOrEmpty().StringOrEmpty("key"))
}

func Test_Post_Timeout(t *testing.T) {
	cf, _ := test.MockCf(t)

	// Setup sample response with delay of 50 ms to fail this call
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		data := gox.StringObjectMap{"status": "ok"}
		_, _ = fmt.Fprintln(w, serialization.StringifySuppressError(data, "{}"))
	}))
	defer ts.Close()

	// Read config and put the port to call
	config := command.Config{}
	err := serialization.ReadYamlFromString(testhelper.TestConfigWithRealServer, &config)
	assert.NoError(t, err)
	config.Servers["testServer"].Port, err = strconv.Atoi(strings.ReplaceAll(ts.URL, "http://127.0.0.1:", ""))
	assert.NoError(t, err)
	config.Apis["delay_timeout_10_POST"].DisableHystrix = true

	// Setup goHttp context
	goxHttpCtx, err := NewGoxHttpContext(cf, &config)
	assert.NoError(t, err)

	// Test 1 - Call http to get data
	ctx, ctxC := context.WithTimeout(context.Background(), 2*time.Second)
	defer ctxC()

	request := command.NewGoxRequestBuilder("delay_timeout_10_POST").
		WithContentTypeJson().
		WithPathParam("id", 1).
		WithBody(gox.StringObjectMap{"key": "value"}).
		WithResponseBuilder(command.NewJsonToObjectResponseBuilder(&gox.StringObjectMap{})).
		Build()
	_, err = goxHttpCtx.Execute(ctx, request)
	assert.Error(t, err)
	if e, ok := err.(*command.GoxHttpError); ok {
		assert.Equal(t, "request_timeout_on_client", e.ErrorCode)
	} else {
		assert.Fail(t, "expected GoxHttpError error")
	}
}

func Test_Post_With_Acceptable_Status_Code(t *testing.T) {
	cf, _ := test.MockCf(t)

	// Setup sample response with delay of 50 ms to fail this call
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if body, err := ioutil.ReadAll(r.Body); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			if _body, err := gox.StringObjectMapFromString(string(body)); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				data := gox.StringObjectMap{"status": "ok"}
				for k, v := range _body {
					data[k] = v
				}
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = fmt.Fprintln(w, serialization.StringifySuppressError(data, "{}"))
			}
		}

	}))
	defer ts.Close()

	// Read config and put the port to call
	config := command.Config{}
	err := serialization.ReadYamlFromString(testhelper.TestConfigWithRealServer, &config)
	assert.NoError(t, err)
	config.Servers["testServer"].Port, err = strconv.Atoi(strings.ReplaceAll(ts.URL, "http://127.0.0.1:", ""))
	assert.NoError(t, err)

	config.Apis["delay_timeout_10_POST"].DisableHystrix = true
	config.Apis["delay_timeout_10_POST"].AcceptableCodes = "202,401"

	// Setup goHttp context
	goxHttpCtx, err := NewGoxHttpContext(cf, &config)
	assert.NoError(t, err)

	// Test 1 - Call http to get data
	ctx, ctxC := context.WithTimeout(context.Background(), 2*time.Second)
	defer ctxC()

	request := command.NewGoxRequestBuilder("delay_timeout_10_POST").
		WithContentTypeJson().
		WithPathParam("id", 1).
		WithBody(gox.StringObjectMap{"key": "value"}).
		WithResponseBuilder(command.NewJsonToObjectResponseBuilder(&gox.StringObjectMap{})).
		Build()
	response, err := goxHttpCtx.Execute(ctx, request)
	assert.NoError(t, err)
	assert.Equal(t, 401, response.StatusCode)
	assert.Equal(t, "ok", response.AsStringObjectMapOrEmpty().StringOrEmpty("status"))
}

func Test_Post_With_Unacceptable_Status_Code(t *testing.T) {
	cf, _ := test.MockCf(t)

	// Setup sample response with delay of 50 ms to fail this call
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := gox.StringObjectMap{"status": "ok"}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write(serialization.ToBytesSuppressError(data))
	}))
	defer ts.Close()

	// Read config and put the port to call
	config := command.Config{}
	err := serialization.ReadYamlFromString(testhelper.TestConfigWithRealServer, &config)
	assert.NoError(t, err)
	config.Servers["testServer"].Port, err = strconv.Atoi(strings.ReplaceAll(ts.URL, "http://127.0.0.1:", ""))
	assert.NoError(t, err)
	config.Apis["delay_timeout_10_POST"].DisableHystrix = true

	// Setup goHttp context
	goxHttpCtx, err := NewGoxHttpContext(cf, &config)
	assert.NoError(t, err)

	// Test 1 - Call http to get data
	ctx, ctxC := context.WithTimeout(context.Background(), 2*time.Second)
	defer ctxC()

	request := command.NewGoxRequestBuilder("delay_timeout_10_POST").
		WithContentTypeJson().
		WithPathParam("id", 1).
		WithResponseBuilder(command.NewJsonToObjectResponseBuilder(&gox.StringObjectMap{})).
		Build()
	_, err = goxHttpCtx.Execute(ctx, request)
	assert.Error(t, err)
}
