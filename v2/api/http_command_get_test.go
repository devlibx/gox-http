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
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func Test_Get_Success(t *testing.T) {
	cf, _ := test.MockCf(t)

	// Setup sample response
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	config.Apis["delay_timeout_10"].DisableHystrix = true

	// Setup goHttp context
	goxHttpCtx, err := NewGoxHttpContext(cf, &config)
	assert.NoError(t, err)

	// Test 1 - Call http to get data
	ctx, ctxC := context.WithTimeout(context.Background(), 2*time.Second)
	defer ctxC()

	request := command.NewGoxRequestBuilder("delay_timeout_10").
		WithContentTypeJson().
		WithPathParam("id", 1).
		WithResponseBuilder(command.NewJsonToObjectResponseBuilder(&gox.StringObjectMap{})).
		Build()
	response, err := goxHttpCtx.Execute(ctx, request)
	assert.NoError(t, err)
	assert.Equal(t, "ok", response.AsStringObjectMapOrEmpty().StringOrEmpty("status"))
}

func Test_Get_Timeout(t *testing.T) {
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
	config.Apis["delay_timeout_10"].DisableHystrix = true

	// Setup goHttp context
	goxHttpCtx, err := NewGoxHttpContext(cf, &config)
	assert.NoError(t, err)

	// Test 1 - Call http to get data
	ctx, ctxC := context.WithTimeout(context.Background(), 2*time.Second)
	defer ctxC()

	request := command.NewGoxRequestBuilder("delay_timeout_10").
		WithContentTypeJson().
		WithPathParam("id", 1).
		WithResponseBuilder(command.NewJsonToObjectResponseBuilder(&gox.StringObjectMap{})).
		Build()
	_, err = goxHttpCtx.Execute(ctx, request)
	assert.Error(t, err)
	if e, ok := err.(*command.GoxHttpError); ok {
		assert.Equal(t, "request_timeout_on_client", e.ErrorCode)
	} else {
		fmt.Println(err)
		assert.Fail(t, "expected GoxHttpError error")
	}
}

func Test_Get_With_Acceptable_Status_Code(t *testing.T) {
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

	config.Apis["delay_timeout_10"].DisableHystrix = true
	config.Apis["delay_timeout_10"].AcceptableCodes = "202,401"

	// Setup goHttp context
	goxHttpCtx, err := NewGoxHttpContext(cf, &config)
	assert.NoError(t, err)

	// Test 1 - Call http to get data
	ctx, ctxC := context.WithTimeout(context.Background(), 2*time.Second)
	defer ctxC()

	request := command.NewGoxRequestBuilder("delay_timeout_10").
		WithContentTypeJson().
		WithPathParam("id", 1).
		WithResponseBuilder(command.NewJsonToObjectResponseBuilder(&gox.StringObjectMap{})).
		Build()
	response, err := goxHttpCtx.Execute(ctx, request)
	assert.NoError(t, err)
	assert.Equal(t, 401, response.StatusCode)
	assert.Equal(t, "ok", response.AsStringObjectMapOrEmpty().StringOrEmpty("status"))
}

func Test_Get_With_Unacceptable_Status_Code(t *testing.T) {
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
	config.Apis["delay_timeout_10"].DisableHystrix = true

	// Setup goHttp context
	goxHttpCtx, err := NewGoxHttpContext(cf, &config)
	assert.NoError(t, err)

	// Test 1 - Call http to get data
	ctx, ctxC := context.WithTimeout(context.Background(), 2*time.Second)
	defer ctxC()

	request := command.NewGoxRequestBuilder("delay_timeout_10").
		WithContentTypeJson().
		WithPathParam("id", 1).
		WithResponseBuilder(command.NewJsonToObjectResponseBuilder(&gox.StringObjectMap{})).
		Build()
	_, err = goxHttpCtx.Execute(ctx, request)
	assert.Error(t, err)
}

func Test_Get_With_Retry(t *testing.T) {
	cf, _ := test.MockCf(t)

	// Setup sample response with delay of 50 ms to fail this call

	var count int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&count, 1)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer ts.Close()

	// Read config and put the port to call
	config := command.Config{}
	err := serialization.ReadYamlFromString(testhelper.TestConfigWithRealServer, &config)
	assert.NoError(t, err)
	config.Servers["testServer"].Port, err = strconv.Atoi(strings.ReplaceAll(ts.URL, "http://127.0.0.1:", ""))
	assert.NoError(t, err)
	config.Apis["delay_timeout_10"].RetryCount = 3
	config.Apis["delay_timeout_10"].Timeout = 10000

	// Setup goHttp context
	goxHttpCtx, err := NewGoxHttpContext(cf, &config)
	assert.NoError(t, err)

	// Test 1 - Call http to get data
	ctx, ctxC := context.WithTimeout(context.Background(), 2*time.Second)
	defer ctxC()

	request := command.NewGoxRequestBuilder("delay_timeout_10").
		WithContentTypeJson().
		WithPathParam("id", 1).
		WithResponseBuilder(command.NewJsonToObjectResponseBuilder(&gox.StringObjectMap{})).
		Build()
	response, err := goxHttpCtx.Execute(ctx, request)
	assert.Error(t, err)
	assert.Equal(t, int32(config.Apis["delay_timeout_10"].RetryCount+1), count)
	assert.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

func Test_Get_With_Retry_With_Finally_Success(t *testing.T) {
	cf, _ := test.MockCf(t)

	// Setup sample response with delay of 50 ms to fail this call

	var count int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&count, 1)
		if count < 3 {
			w.WriteHeader(http.StatusUnauthorized)
		} else {
			w.WriteHeader(http.StatusOK)
			data := gox.StringObjectMap{"status": "ok"}
			_, _ = fmt.Fprintln(w, serialization.StringifySuppressError(data, "{}"))
		}
	}))
	defer ts.Close()

	// Read config and put the port to call
	config := command.Config{}
	err := serialization.ReadYamlFromString(testhelper.TestConfigWithRealServer, &config)
	assert.NoError(t, err)
	config.Servers["testServer"].Port, err = strconv.Atoi(strings.ReplaceAll(ts.URL, "http://127.0.0.1:", ""))
	assert.NoError(t, err)
	config.Apis["delay_timeout_10"].RetryCount = 3
	config.Apis["delay_timeout_10"].Timeout = 10000

	// Setup goHttp context
	goxHttpCtx, err := NewGoxHttpContext(cf, &config)
	assert.NoError(t, err)

	// Test 1 - Call http to get data
	ctx, ctxC := context.WithTimeout(context.Background(), 2*time.Second)
	defer ctxC()

	request := command.NewGoxRequestBuilder("delay_timeout_10").
		WithContentTypeJson().
		WithPathParam("id", 1).
		WithResponseBuilder(command.NewJsonToObjectResponseBuilder(&gox.StringObjectMap{})).
		Build()
	response, err := goxHttpCtx.Execute(ctx, request)
	assert.NoError(t, err)
	assert.Equal(t, int32(3), count)
	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, "ok", response.AsStringObjectMapOrEmpty().StringOrEmpty("status"))
}

func Test_Get_With_Retry_Non_2xx_But_Acceptable_Code(t *testing.T) {
	cf, _ := test.MockCf(t)

	// Setup sample response with delay of 50 ms to fail this call

	var count int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&count, 1)
		w.WriteHeader(http.StatusUnauthorized)
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
	config.Apis["delay_timeout_10"].RetryCount = 3
	config.Apis["delay_timeout_10"].Timeout = 10000
	config.Apis["delay_timeout_10"].AcceptableCodes = "200, 401"

	// Setup goHttp context
	goxHttpCtx, err := NewGoxHttpContext(cf, &config)
	assert.NoError(t, err)

	// Test 1 - Call http to get data
	ctx, ctxC := context.WithTimeout(context.Background(), 2*time.Second)
	defer ctxC()

	request := command.NewGoxRequestBuilder("delay_timeout_10").
		WithContentTypeJson().
		WithPathParam("id", 1).
		WithResponseBuilder(command.NewJsonToObjectResponseBuilder(&gox.StringObjectMap{})).
		Build()
	response, err := goxHttpCtx.Execute(ctx, request)
	assert.NoError(t, err)
	assert.Equal(t, int32(1), count)
	assert.Equal(t, http.StatusUnauthorized, response.StatusCode)
	assert.Equal(t, "ok", response.AsStringObjectMapOrEmpty().StringOrEmpty("status"))
}
