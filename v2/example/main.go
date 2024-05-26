package main

import (
	"context"
	"github.com/devlibx/gox-base"
	"github.com/devlibx/gox-base/serialization"
	goxHttpApi "github.com/devlibx/gox-http/v2/api"
	"github.com/devlibx/gox-http/v2/command"
	"log/slog"
)

// Here you can define your own configuration
// We have used "jsonplaceholder" as a test server. A api "getPosts" is defined which uses "server=jsonplaceholder"
var httpConfig = `
servers:
  jsonplaceholder:
    host: jsonplaceholder.typicode.com
    port: 443
    https: true
    connect_timeout: 1000
    connection_request_timeout: 1000
  
apis: 
  getPosts:
    method: GET
    path: /posts/{id}
    server: jsonplaceholder
    timeout: 1000
    acceptable_codes: 200,201
`

func main() {

	// Read config and
	config := command.Config{}
	err := serialization.ReadYamlFromString(httpConfig, &config)
	if err != nil {
		slog.Error("got error in reading config", err)
		return
	}

	// Setup goHttp context
	goxHttpCtx, err := goxHttpApi.NewGoxHttpContext(gox.NewCrossFunction(), &config)
	if err != nil {
		slog.Error("got error in creating gox http context config", err)
		return
	}

	// Execute HTTP request
	if successResponse, err := goxHttpApi.ExecuteHttp[successPojo, errorPojo](
		context.Background(),
		goxHttpCtx,
		command.NewGoxRequestBuilder("getPosts").
			WithContentTypeJson().
			WithPathParam("id", 1).
			Build(),
	); err != nil {

		// If you expect some error body in case api returns error then you can get it from
		// error itself = actually this is just a wrapper of errors.As(err, out) method
		//
		// errorResponsePayloadExists = true if there is some error payload
		// ok = true (we should never get ok = false if you passed the error returned from ExecuteHttp method)
		errorResponse, errorResponsePayloadExists, ok := goxHttpApi.ExtractError[errorPojo](err)
		_, _ = errorResponsePayloadExists, ok

		slog.Error("[Failed to execute]", slog.Any("status", errorResponse.Response))
	} else {
		slog.Info("Got result", slog.Int("id", successResponse.Response.Id), slog.String("name", successResponse.Response.Name))
		// Response
		// 2024/05/27 01:32:13 INFO Got result id=1 name="sunt aut facere repellat provident occaecati excepturi optio reprehenderit"
	}
}

type successPojo struct {
	Id   int    `json:"id"`
	Name string `json:"title"`
}

type errorPojo struct {
	Status string `json:"status"`
}
