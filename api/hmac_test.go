package goxHttpApi

import (
	"context"
	"fmt"
	"github.com/devlibx/gox-base/v2"
	"github.com/devlibx/gox-base/v2/serialization"
	"github.com/devlibx/gox-base/v2/test"
	"github.com/devlibx/gox-http/v4/command"
	"github.com/devlibx/gox-http/v4/interceptor"
	"github.com/devlibx/gox-http/v4/testhelper"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func Test_Get_Success_With_Hmac(t *testing.T) {
	cf, _ := test.MockCf(t)

	// Define the layout
	layout := "2006-01-02 15:04:05"
	timeString := "2024-01-01 00:00:00"

	// Parse the date and time string into a time.Time object
	timeNow, err := time.Parse(layout, timeString)
	assert.NoError(t, err)

	// Setup sample response
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := gox.StringObjectMap{"status": "ok"}
		_, _ = fmt.Fprintln(w, serialization.StringifySuppressError(data, "{}"))
		assert.Equal(t, "YyfbKp/6v0IrWPtdLYDMY6WYv+kKg5wv4bE89EOK/jw=", r.Header["X-Hash-Code"][0])
		assert.Equal(t, "1704067200000", r.Header["X-Timestamp"][0])
	}))
	defer ts.Close()

	// Read config and put the port to call
	config := command.Config{}
	err = serialization.ReadYamlFromString(testhelper.TestConfigWithRealServer, &config)
	assert.NoError(t, err)
	config.Servers["testServer"].Port, err = strconv.Atoi(strings.ReplaceAll(ts.URL, "http://127.0.0.1:", ""))
	assert.NoError(t, err)
	config.Apis["delay_timeout_10"].DisableHystrix = true

	config.Servers["testServer"].InterceptorConfig = &interceptor.Config{
		HmacConfig: &interceptor.HmacConfig{
			Key:                          "secret_123",
			DumpDebug:                    true,
			HashHeaderKey:                "X-Hash-Code",
			TimestampHeaderKey:           "X-Timestamp",
			HeadersToIncludeInSignature:  []string{"X-Header-1", "X-Header-2"},
			ConvertHeaderKeysToLowerCase: true,
		},
	}

	// Setup goHttp context
	goxHttpCtx, err := NewGoxHttpContext(cf, &config)
	assert.NoError(t, err)

	// Test 1 - Call http to get data
	ctx, ctxC := context.WithTimeout(context.Background(), 2*time.Second)
	defer ctxC()
	ctx = context.WithValue(ctx, "__testing_ts__", fmt.Sprintf("%d", timeNow.UnixMilli()))

	request := command.NewGoxRequestBuilder("delay_timeout_10").
		WithContentTypeJson().
		WithHeader("X-Header-1", 101).
		WithHeader("X-Header-2", "header2").
		WithPathParam("id", 1).
		WithResponseBuilder(command.NewJsonToObjectResponseBuilder(&gox.StringObjectMap{})).
		Build()
	response, err := goxHttpCtx.Execute(ctx, request)
	assert.NoError(t, err)
	assert.Equal(t, "ok", response.AsStringObjectMapOrEmpty().StringOrEmpty("status"))
}

func Test_Post_Success_With_Hmac(t *testing.T) {
	cf, _ := test.MockCf(t)

	// Define the layout
	layout := "2006-01-02 15:04:05"
	timeString := "2024-01-01 00:00:00"

	// Parse the date and time string into a time.Time object
	timeNow, err := time.Parse(layout, timeString)
	assert.NoError(t, err)

	// Setup sample response
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := gox.StringObjectMap{"status": "ok"}
		_, _ = fmt.Fprintln(w, serialization.StringifySuppressError(data, "{}"))
		assert.Equal(t, "RWTd7uSqc1JrQEwJcFsyxA85qybw0MsVZCwKnT9Sgos=", r.Header["X-Hash-Code"][0])
		assert.Equal(t, "1704067200000", r.Header["X-Timestamp"][0])
	}))
	defer ts.Close()

	// Read config and put the port to call
	config := command.Config{}
	err = serialization.ReadYamlFromString(testhelper.TestConfigWithRealServer, &config)
	assert.NoError(t, err)
	config.Servers["testServer"].Port, err = strconv.Atoi(strings.ReplaceAll(ts.URL, "http://127.0.0.1:", ""))
	assert.NoError(t, err)
	config.Apis["delay_timeout_10"].DisableHystrix = true

	config.Servers["testServer"].InterceptorConfig = &interceptor.Config{
		HmacConfig: &interceptor.HmacConfig{
			Key:                          "secret_123",
			DumpDebug:                    true,
			HashHeaderKey:                "X-Hash-Code",
			TimestampHeaderKey:           "X-Timestamp",
			HeadersToIncludeInSignature:  []string{"X-Header-1", "X-Header-2"},
			ConvertHeaderKeysToLowerCase: true,
		},
	}

	// Setup goHttp context
	goxHttpCtx, err := NewGoxHttpContext(cf, &config)
	assert.NoError(t, err)

	// Test 1 - Call http to get data
	ctx, ctxC := context.WithTimeout(context.Background(), 2*time.Second)
	defer ctxC()
	ctx = context.WithValue(ctx, "__testing_ts__", fmt.Sprintf("%d", timeNow.UnixMilli()))

	request := command.NewGoxRequestBuilder("delay_timeout_10").
		WithContentTypeJson().
		WithHeader("X-Header-1", 101).
		WithHeader("X-Header-2", "header2").
		WithPathParam("id", 1).
		WithBody(`{"status": "ok"}`).
		WithResponseBuilder(command.NewJsonToObjectResponseBuilder(&gox.StringObjectMap{})).
		Build()
	response, err := goxHttpCtx.Execute(ctx, request)
	assert.NoError(t, err)
	assert.Equal(t, "ok", response.AsStringObjectMapOrEmpty().StringOrEmpty("status"))
}

func Test_Post_Success_With_Hmac_Struct(t *testing.T) {
	cf, _ := test.MockCf(t)

	// Define the layout
	layout := "2006-01-02 15:04:05"
	timeString := "2024-01-01 00:00:00"

	// Parse the date and time string into a time.Time object
	timeNow, err := time.Parse(layout, timeString)
	assert.NoError(t, err)

	// Setup sample response
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := gox.StringObjectMap{"status": "ok"}
		_, _ = fmt.Fprintln(w, serialization.StringifySuppressError(data, "{}"))
		assert.Equal(t, "bux/MHZnySHNHySSzvaRcE4fKzXSwmvPcTAO31rs61I=", r.Header["X-Hash-Code"][0])
		assert.Equal(t, "1704067200000", r.Header["X-Timestamp"][0])
	}))
	defer ts.Close()

	// Read config and put the port to call
	config := command.Config{}
	err = serialization.ReadYamlFromString(testhelper.TestConfigWithRealServer, &config)
	assert.NoError(t, err)
	config.Servers["testServer"].Port, err = strconv.Atoi(strings.ReplaceAll(ts.URL, "http://127.0.0.1:", ""))
	assert.NoError(t, err)
	config.Apis["delay_timeout_10"].DisableHystrix = true

	config.Servers["testServer"].InterceptorConfig = &interceptor.Config{
		HmacConfig: &interceptor.HmacConfig{
			Key:                          "secret_123",
			DumpDebug:                    true,
			HashHeaderKey:                "X-Hash-Code",
			TimestampHeaderKey:           "X-Timestamp",
			HeadersToIncludeInSignature:  []string{"X-Header-1", "X-Header-2"},
			ConvertHeaderKeysToLowerCase: true,
		},
	}

	// Setup goHttp context
	goxHttpCtx, err := NewGoxHttpContext(cf, &config)
	assert.NoError(t, err)

	// Test 1 - Call http to get data
	ctx, ctxC := context.WithTimeout(context.Background(), 2*time.Second)
	defer ctxC()
	ctx = context.WithValue(ctx, "__testing_ts__", fmt.Sprintf("%d", timeNow.UnixMilli()))

	type req struct {
		Status string `json:"status"`
	}
	request := command.NewGoxRequestBuilder("delay_timeout_10").
		WithContentTypeJson().
		WithHeader("X-Header-1", 101).
		WithHeader("X-Header-2", "header2").
		WithPathParam("id", 1).
		WithBody(req{Status: "ok"}).
		WithResponseBuilder(command.NewJsonToObjectResponseBuilder(&gox.StringObjectMap{})).
		Build()
	response, err := goxHttpCtx.Execute(ctx, request)
	assert.NoError(t, err)
	assert.Equal(t, "ok", response.AsStringObjectMapOrEmpty().StringOrEmpty("status"))
}
