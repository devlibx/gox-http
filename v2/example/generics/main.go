package main

import (
	"context"
	"github.com/devlibx/gox-base"
	"github.com/devlibx/gox-base/serialization"
	goxHttpApi "github.com/devlibx/gox-http/v2/api"
	"github.com/devlibx/gox-http/v2/command"
	"log"
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
  testServer:
    host: localhost
    port: 9123

apis:
  getPosts:
    method: GET
    path: /posts/{id}
    server: jsonplaceholder
    timeout: 1000
    acceptable_codes: 200,201
  delay_timeout_10:
    path: /delay
    server: testServer
    timeout: 10
    concurrency: 3 
  delay_10_ms:
    path: /delay/10_ms
    server: testServer
    timeout: 100
    concurrency: 300
`

func main() {
	cf := gox.NewCrossFunction()

	// Read config and
	config := command.Config{}
	err := serialization.ReadYamlFromString(httpConfig, &config)
	if err != nil {
		slog.Error("got error in reading config", err)
		return
	}

	// Setup goHttp context
	goxHttpCtx, err := goxHttpApi.NewGoxHttpContext(cf, &config)
	if err != nil {
		slog.Error("got error in creating gox http context config", err)
		return
	}

	request := command.NewGoxRequestBuilder("getPosts").
		WithContentTypeJson().
		WithContentTypeJson().
		WithPathParam("id", 1).
		WithResponseBuilder(command.NewJsonToObjectResponseBuilder(&gox.StringObjectMap{})).
		Build()

	result, err := goxHttpCtx.Execute(context.Background(), "getPosts", request)
	if err != nil {
		slog.Error("Failed to execute", err)
		return
	}

	log.Println("Got result", result)
}
