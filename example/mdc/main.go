package main

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/devlibx/gox-base/v2"
	"github.com/devlibx/gox-base/v2/serialization"
	goxHttpApi "github.com/devlibx/gox-http/v4/api"
	"github.com/devlibx/gox-http/v4/command"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
	"time"
)

//go:embed sample.yaml
var httpConfig string

func main() {

	go func() {
		router := gin.New()
		router.POST("/example", func(c *gin.Context) {
			fmt.Println("x-request-id=", c.GetHeader("x-request-id"))
			fmt.Println("x-user-id=", c.GetHeader("x-user-id"))
			c.Status(http.StatusOK)
		})
		router.Run(":8018")
	}()
	time.Sleep(1 * time.Second)

	// Read config and
	config := command.Config{}
	err := serialization.ReadYamlFromString(httpConfig, &config)
	if err != nil {
		slog.Error("got error in reading config", "error", err)
		return
	}

	// Setup goHttp context
	goxHttpCtx, err := goxHttpApi.NewGoxHttpContext(gox.NewCrossFunction(), &config)
	if err != nil {
		slog.Error("got error in creating gox http context config", "error", err)
		return
	}

	successResponse, err := goxHttpApi.ExecuteHttp[any, any](
		context.Background(),
		goxHttpCtx,
		command.NewGoxRequestBuilder("example").
			WithContentTypeJson().
			WithHeader("x-request-id", "1234").
			WithHeader("x-user-id", "user_1").
			Build(),
	)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "x-request-id", "1234")
	ctx = context.WithValue(ctx, "x-user-id", "user_1")
	successResponse, err = goxHttpApi.ExecuteHttp[any, any](
		ctx,
		goxHttpCtx,
		command.NewGoxRequestBuilder("example").
			WithContentTypeJson().
			Build(),
	)
	_, _ = successResponse, err
}
