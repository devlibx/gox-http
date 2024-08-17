package goxHttpApi

import (
	"context"
	"fmt"
	"github.com/afex/hystrix-go/hystrix"
	"github.com/devlibx/gox-base/v2"
	"github.com/devlibx/gox-base/v2/serialization"
	"github.com/devlibx/gox-base/v2/test"
	"github.com/devlibx/gox-http/v2/command"
	httpCommand "github.com/devlibx/gox-http/v2/command/http"
	"github.com/devlibx/gox-http/v2/testhelper"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func Test_Hystrix_Update(t *testing.T) {

	// defer goleak.VerifyNone(t)
	cf, _ := test.MockCf(t)
	httpCommand.HystrixConfigMap = gox.StringObjectMap{}
	hystrix.Flush()

	// Setup sample response
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := gox.StringObjectMap{"status": "ok", "url": r.URL.String()}
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
	assert.Equal(t, "/delay", response.AsStringObjectMapOrEmpty().StringOrEmpty("url"))

	config.Apis["delay_timeout_10"].Path = "/delay_new"
	err = goxHttpCtx.ReloadApi("delay_timeout_10")
	assert.NoError(t, err)

	// Test 2 - Call http to get data
	ctx, ctxC = context.WithTimeout(context.Background(), 2*time.Second)
	defer ctxC()

	request = command.NewGoxRequestBuilder("delay_timeout_10").
		WithContentTypeJson().
		WithPathParam("id", 1).
		WithResponseBuilder(command.NewJsonToObjectResponseBuilder(&gox.StringObjectMap{})).
		Build()
	response, err = goxHttpCtx.Execute(ctx, request)
	assert.NoError(t, err)
	assert.Equal(t, "ok", response.AsStringObjectMapOrEmpty().StringOrEmpty("status"))
	assert.Equal(t, "/delay_new", response.AsStringObjectMapOrEmpty().StringOrEmpty("url"))

	config.Apis["new_api"] = &command.Api{
		Name:        "new_api",
		Method:      "GET",
		Path:        "/bad_new",
		Server:      "testServer",
		Timeout:     100,
		Concurrency: 10,
		QueueSize:   10,
	}
	err = goxHttpCtx.ReloadApi("new_api")
	assert.NoError(t, err)

	// Test 2 - Call http to get data
	ctx, ctxC = context.WithTimeout(context.Background(), 2*time.Second)
	defer ctxC()

	request = command.NewGoxRequestBuilder("new_api").
		WithContentTypeJson().
		WithPathParam("id", 1).
		WithResponseBuilder(command.NewJsonToObjectResponseBuilder(&gox.StringObjectMap{})).
		Build()
	response, err = goxHttpCtx.Execute(ctx, request)
	assert.NoError(t, err)
	assert.Equal(t, "ok", response.AsStringObjectMapOrEmpty().StringOrEmpty("status"))
	assert.Equal(t, "/bad_new", response.AsStringObjectMapOrEmpty().StringOrEmpty("url"))
}
