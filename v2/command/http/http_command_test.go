package httpCommand

import (
	"context"
	"fmt"
	"github.com/devlibx/gox-base"
	"github.com/devlibx/gox-base/test"
	"github.com/devlibx/gox-http/v2/command"
	"github.com/devlibx/gox-http/v2/testhelper"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestHttpCommand_Sync(t *testing.T) {
	cf, _ := test.MockCf(t)

	config := command.Config{}
	err := testhelper.GetTestConfig(&config)
	assert.NoError(t, err)

	server, err := config.FindServerByName("jsonplaceholder")
	assert.NoError(t, err)

	api, err := config.FindApiByName("getPosts")
	assert.NoError(t, err)

	httpCmd, err := NewHttpCommand(cf, server, api)
	assert.NoError(t, err)

	result, err := httpCmd.Execute(context.TODO(), &command.GoxRequest{
		PathParam:       map[string][]string{"id": {"1"}},
		ResponseBuilder: command.NewJsonToObjectResponseBuilder(&gox.StringObjectMap{}),
	})
	assert.NoError(t, err)
	fmt.Println(result.Response)
}

func TestHttpCommand_Async(t *testing.T) {
	cf, _ := test.MockCf(t)

	config := command.Config{}
	err := testhelper.GetTestConfig(&config)
	assert.NoError(t, err)

	server, err := config.FindServerByName("jsonplaceholder")
	assert.NoError(t, err)

	api, err := config.FindApiByName("getPosts")
	assert.NoError(t, err)

	httpCmd, err := NewHttpCommand(cf, server, api)
	assert.NoError(t, err)

	ctx, ctxCan := context.WithTimeout(context.Background(), 5*time.Second)
	defer ctxCan()
	result := httpCmd.ExecuteAsync(ctx, &command.GoxRequest{
		PathParam:       map[string][]string{"id": {"1"}},
		ResponseBuilder: command.NewJsonToObjectResponseBuilder(&gox.StringObjectMap{}),
	})

	select {
	case <-ctx.Done():
		assert.Fail(t, "context timeout")
	case r := <-result:
		assert.NoError(t, r.Err)
		fmt.Println(r.Response)
	}
}

func TestBuilder(t *testing.T) {
	cf, _ := test.MockCf(t)

	config := command.Config{}
	err := testhelper.GetTestConfig(&config)
	assert.NoError(t, err)

	server, err := config.FindServerByName("jsonplaceholder")
	assert.NoError(t, err)

	api, err := config.FindApiByName("getPosts")
	assert.NoError(t, err)

	httpCmd, err := NewHttpCommand(cf, server, api)
	assert.NoError(t, err)

	request := command.NewGoxRequestBuilder("getPosts").
		WithContentTypeJson().
		WithPathParam("id", 1).
		WithResponseBuilder(command.NewJsonToObjectResponseBuilder(&gox.StringObjectMap{})).
		Build()

	result, err := httpCmd.Execute(context.TODO(), request)
	assert.NoError(t, err)
	fmt.Println(result.Response)
}
