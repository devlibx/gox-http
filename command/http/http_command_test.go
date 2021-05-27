package httpCommand

import (
	"context"
	"fmt"
	"github.com/devlibx/gox-base"
	"github.com/devlibx/gox-base/test"
	"github.com/devlibx/gox-http/command"
	"github.com/devlibx/gox-http/testData"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestHttpCommand_Sync(t *testing.T) {
	cf, _ := test.MockCf(t)

	config := command.Config{}
	err := testData.GetTestConfig(&config)
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
	err := testData.GetTestConfig(&config)
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
