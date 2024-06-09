package goxHttpApi

import (
	"context"
	"fmt"
	"github.com/afex/hystrix-go/hystrix"
	"github.com/devlibx/gox-base"
	"github.com/devlibx/gox-base/serialization"
	"github.com/devlibx/gox-base/test"
	"github.com/devlibx/gox-http/v2/command"
	httpCommand "github.com/devlibx/gox-http/v2/command/http"
	"github.com/devlibx/gox-http/v2/testhelper"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func Test_Hystrix_Get_Success(t *testing.T) {
	// defer goleak.VerifyNone(t)
	cf, _ := test.MockCf(t)
	httpCommand.HystrixConfigMap = gox.StringObjectMap{}
	hystrix.Flush()

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

func Test_Hystrix_Get_Timeout_WhenHttpCallTimeoutFirst(t *testing.T) {
	// defer goleak.VerifyNone(t)
	cf, _ := test.MockCf(t)
	httpCommand.HystrixConfigMap = gox.StringObjectMap{}
	hystrix.Flush()

	// hack hystrix command to have high timeout
	httpCommand.HystrixConfigMap["delay_timeout_10"] = gox.StringObjectMap{"timeout": 100}

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

func Test_Hystrix_Get_Timeout_WhenHystrixTimeoutHappensBeforeHttpTimeout(t *testing.T) {
	// defer goleak.VerifyNone(t)
	cf, _ := test.MockCf(t)
	httpCommand.HystrixConfigMap = gox.StringObjectMap{}
	hystrix.Flush()

	// hack hystrix command to have ver low timeout
	httpCommand.HystrixConfigMap["delay_timeout_10"] = gox.StringObjectMap{"timeout": 1}

	// Setup sample response with delay of 50 ms to fail this call
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	config.Apis["delay_timeout_10"].Timeout = 100

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
	assert.Error(t, err, "did not expected error")
	if e, ok := err.(*command.GoxHttpError); ok {
		assert.Equal(t, "hystrix_timeout", e.ErrorCode)
	} else {
		fmt.Println(err)
		assert.Fail(t, "expected GoxHttpError error")
	}
}

func Test_Hystrix_Get_With_Acceptable_Status_Code(t *testing.T) {
	// defer goleak.VerifyNone(t)
	cf, _ := test.MockCf(t)
	httpCommand.HystrixConfigMap = gox.StringObjectMap{}
	hystrix.Flush()

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

func Test_Hystrix_Get_With_Unacceptable_Status_Code(t *testing.T) {
	// defer goleak.VerifyNone(t)
	cf, _ := test.MockCf(t)
	httpCommand.HystrixConfigMap = gox.StringObjectMap{}
	hystrix.Flush()

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

func Test_Hystrix_Get_Verify_Circuit_Will_Open_On_Too_ManyErrors(t *testing.T) {
	// defer goleak.VerifyNone(t)
	cf, _ := test.MockCf(t)
	httpCommand.HystrixConfigMap = gox.StringObjectMap{}
	hystrix.Flush()

	// Setup sample response with delay of 50 ms to fail this call
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	// Setup goHttp context
	goxHttpCtx, err := NewGoxHttpContext(cf, &config)
	assert.NoError(t, err)

	// Test 1 - Call http to get data
	ctx, ctxC := context.WithTimeout(context.Background(), 2*time.Second)
	defer ctxC()

	errorCount := int32(0)
	for i := 0; i < 1000; i++ {
		request := command.NewGoxRequestBuilder("delay_timeout_10").
			WithContentTypeJson().
			WithPathParam("id", 1).
			WithResponseBuilder(command.NewJsonToObjectResponseBuilder(&gox.StringObjectMap{})).
			Build()
		_, err = goxHttpCtx.Execute(ctx, request)
		assert.Error(t, err)
		if e, ok := err.(*command.GoxHttpError); ok {
			if e.IsHystrixCircuitOpenError() {
				atomic.AddInt32(&errorCount, 1)
			}
		} else {
			assert.Fail(t, "expected error as command.GoxHttpError")
		}
	}
	assert.True(t, errorCount > 100)
	fmt.Println(errorCount)
}

func Test_Hystrix_Get_Verify_Circuit_Will_Open_Due_To_Timeout(t *testing.T) {
	// defer goleak.VerifyNone(t)
	cf, _ := test.MockCf(t)
	httpCommand.HystrixConfigMap = gox.StringObjectMap{}
	hystrix.Flush()

	// Setup sample response with delay of 50 ms to fail this call
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		data := gox.StringObjectMap{"status": "ok"}
		_, _ = fmt.Fprintln(w, serialization.StringifySuppressError(data, "{}"))
	}))
	defer ts.Close()

	httpCommand.HystrixConfigMap["delay_timeout_10"] = gox.StringObjectMap{"timeout": 1}

	// Read config and put the port to call
	config := command.Config{}
	err := serialization.ReadYamlFromString(testhelper.TestConfigWithRealServer, &config)
	assert.NoError(t, err)
	config.Servers["testServer"].Port, err = strconv.Atoi(strings.ReplaceAll(ts.URL, "http://127.0.0.1:", ""))
	assert.NoError(t, err)
	config.Apis["delay_timeout_10"].Timeout = 100

	// Setup goHttp context
	goxHttpCtx, err := NewGoxHttpContext(cf, &config)
	assert.NoError(t, err)

	// Test 1 - Call http to get data
	ctx, ctxC := context.WithTimeout(context.Background(), 2*time.Second)
	defer ctxC()

	errorCountDueToCircuitOpen := int32(0)
	errorCountDueToHystrixTimeout := int32(0)
	for i := 0; i < 1000; i++ {
		request := command.NewGoxRequestBuilder("delay_timeout_10").
			WithContentTypeJson().
			WithPathParam("id", 1).
			WithResponseBuilder(command.NewJsonToObjectResponseBuilder(&gox.StringObjectMap{})).
			Build()
		_, err = goxHttpCtx.Execute(ctx, request)
		assert.Error(t, err)
		if e, ok := err.(*command.GoxHttpError); ok {
			if e.IsHystrixCircuitOpenError() {
				atomic.AddInt32(&errorCountDueToCircuitOpen, 1)
			} else if e.IsHystrixTimeoutError() {
				atomic.AddInt32(&errorCountDueToHystrixTimeout, 1)
			}
		} else {
			assert.Fail(t, "expected error as command.GoxHttpError")
		}
	}
	assert.True(t, errorCountDueToCircuitOpen > 100)
	assert.True(t, errorCountDueToHystrixTimeout > 1)
	fmt.Println("errorCountDueToCircuitOpen=", errorCountDueToCircuitOpen, "errorCountDueToHystrixTimeout=", errorCountDueToHystrixTimeout)
}

func Test_Hystrix_Get_Verify_We_Go_Too_Many_Requests(t *testing.T) {
	// defer goleak.VerifyNone(t)
	cf, _ := test.MockCf(t)
	httpCommand.HystrixConfigMap = gox.StringObjectMap{}
	hystrix.Flush()

	// Setup sample response with delay of 50 ms to fail this call
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		data := gox.StringObjectMap{"status": "ok"}
		_, _ = fmt.Fprintln(w, serialization.StringifySuppressError(data, "{}"))
	}))
	defer ts.Close()

	httpCommand.HystrixConfigMap["delay_timeout_10"] = gox.StringObjectMap{"timeout": 100}

	// Read config and put the port to call
	config := command.Config{}
	err := serialization.ReadYamlFromString(testhelper.TestConfigWithRealServer, &config)
	assert.NoError(t, err)
	config.Servers["testServer"].Port, err = strconv.Atoi(strings.ReplaceAll(ts.URL, "http://127.0.0.1:", ""))
	assert.NoError(t, err)
	config.Apis["delay_timeout_10"].Concurrency = 10
	config.Apis["delay_timeout_10"].Timeout = 100

	// Setup goHttp context
	goxHttpCtx, err := NewGoxHttpContext(cf, &config)
	assert.NoError(t, err)

	// Test 1 - Call http to get data
	ctx, ctxC := context.WithTimeout(context.Background(), 20*time.Second)
	defer ctxC()

	sg := sync.WaitGroup{}
	errorCountDueToReject := int32(0)
	_ = errorCountDueToReject
	for i := 0; i < 1000; i++ {
		sg.Add(1)
		go func() {
			request := command.NewGoxRequestBuilder("delay_timeout_10").
				WithContentTypeJson().
				WithPathParam("id", 1).
				WithResponseBuilder(command.NewJsonToObjectResponseBuilder(&gox.StringObjectMap{})).
				Build()
			_, err = goxHttpCtx.Execute(ctx, request)
			if err != nil {
				if e, ok := err.(*command.GoxHttpError); ok {
					if e.IsHystrixRejectedError() {
						atomic.AddInt32(&errorCountDueToReject, 1)
					}
				}
			}
			sg.Done()
		}()
	}
	sg.Wait()
	assert.True(t, errorCountDueToReject > 2)
	fmt.Println("errorCountDueToReject", errorCountDueToReject)
}
