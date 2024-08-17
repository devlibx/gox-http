package main

import (
	"context"
	"fmt"
	"github.com/devlibx/gox-base/v2"
	"github.com/devlibx/gox-base/v2/serialization"
	goxHttpApi "github.com/devlibx/gox-http/v2/api"
	"github.com/devlibx/gox-http/v2/command"
	"github.com/devlibx/gox-http/v2/example/perf/helper"
	"github.com/go-resty/resty/v2"
	dwMetric "github.com/rcrowley/go-metrics"
	"log"
	"os"
	"time"
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
	// run perf test
	if true {
		// perfMainWithGoxHttp()
		// perfMainWithResty()
		// return
	}

	cf := gox.NewCrossFunction()

	// Read config and
	config := command.Config{}
	err := serialization.ReadYamlFromString(httpConfig, &config)
	if err != nil {
		log.Println("got error in reading config", err)
		return
	}

	// Setup goHttp context
	goxHttpCtx, err := goxHttpApi.NewGoxHttpContext(cf, &config)
	if err != nil {
		log.Println("got error in creating gox http context config", err)
		return
	}

	// Make a http call and get the result
	// 	ResponseBuilder - this is used to convert json response to your custom object
	//
	//  The following interface can be implemented to convert from bytes to the desired output.
	//  response.Response will hold the object which is returned from  ResponseBuilder
	//
	//	type ResponseBuilder interface {
	//		Response(data []byte) (interface{}, error)
	//	}
	request := command.NewGoxRequestBuilder("getPosts").
		WithContentTypeJson().
		WithPathParam("id", 1).
		WithResponseBuilder(command.NewJsonToObjectResponseBuilder(&gox.StringObjectMap{})).
		Build()
	response, err := goxHttpCtx.Execute(context.Background(), request)
	if err != nil {

		// Error details can be extracted from *command.GoxHttpError
		if goxError, ok := err.(*command.GoxHttpError); ok {
			if goxError.Is5xx() {
				fmt.Println("got 5xx error")
			} else if goxError.Is4xx() {
				fmt.Println("got 5xx error")
			} else if goxError.IsBadRequest() {
				fmt.Println("got bad request error")
			} else if goxError.IsHystrixCircuitOpenError() {
				fmt.Println("hystrix circuit is open due to many errors")
			} else if goxError.IsHystrixTimeoutError() {
				fmt.Println("hystrix timeout because http call took longer then configured")
			} else if goxError.IsHystrixRejectedError() {
				fmt.Println("hystrix rejected the request because too many concurrent request are made")
			} else if goxError.IsHystrixError() {
				fmt.Println("hystrix error - timeout/circuit open/rejected")
			}
		} else {
			fmt.Println("got unknown error")
		}

	} else {
		fmt.Println(serialization.Stringify(response.Response))
		// {some json response ...}
	}
}

func perfMainWithGoxHttp() {
	counter := dwMetric.NewCounter()
	errCounter := dwMetric.NewCounter()
	dwMetric.Register("success", counter)
	dwMetric.Register("error", errCounter)

	// Start the mock server
	ctx, cancelFunc := context.WithTimeout(context.TODO(), time.Duration(10*time.Second))
	defer cancelFunc()
	overChannel, err := helper.StartMockServer(ctx, 9123)
	if err != nil {
		panic(err)
	}
	_ = overChannel

	cf := gox.NewCrossFunction()

	// Read config and
	config := command.Config{}
	err = serialization.ReadYamlFromString(httpConfig, &config)
	if err != nil {
		log.Println("got error in reading config", err)
		return
	}

	// Setup goHttp context
	goxHttpCtx, err := goxHttpApi.NewGoxHttpContext(cf, &config)
	if err != nil {
		log.Println("got error in creating gox http context config", err)
		return
	}

	for i := 1; i < 100; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
				// No-OP
				default:
					makeGoxHttpCall(goxHttpCtx, counter, errCounter)
				}
			}
		}()
	}

	go func() {
		go dwMetric.Log(dwMetric.DefaultRegistry, 1*time.Second, log.New(os.Stdout, "metrics: ", log.Lmicroseconds))
	}()

	go func() {
		time.Sleep(10 * time.Second)
		cancelFunc()
	}()
	<-overChannel
}

func makeGoxHttpCall(goxHttpCtx goxHttpApi.GoxHttpContext, counter dwMetric.Counter, errCounter dwMetric.Counter) {
	request := command.NewGoxRequestBuilder("delay_10_ms").
		WithContentTypeJson().
		WithPathParam("id", 1).
		WithResponseBuilder(command.NewJsonToObjectResponseBuilder(&gox.StringObjectMap{})).
		Build()
	response, err := goxHttpCtx.Execute(context.Background(), request)
	if err == nil {
		counter.Inc(1)
	} else {
		errCounter.Inc(1)
	}
	_ = response
	// fmt.Println(response)
}

func perfMainWithResty() {
	counter := dwMetric.NewCounter()
	errCounter := dwMetric.NewCounter()
	dwMetric.Register("success", counter)
	dwMetric.Register("error", errCounter)

	// Start the mock server
	ctx, cancelFunc := context.WithTimeout(context.TODO(), time.Duration(100*time.Second))
	defer cancelFunc()
	overChannel, err := helper.StartMockServer(ctx, 9123)
	if err != nil {
		panic(err)
	}
	_ = overChannel

	restyClient := resty.New()
	restyClient.SetHostURL("http://localhost:9123")

	for i := 1; i < 100; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
				// No-OP
				default:
					makeRestyCall(restyClient, counter, errCounter)
				}
			}
		}()
	}

	go func() {
		go dwMetric.Log(dwMetric.DefaultRegistry, 1*time.Second, log.New(os.Stdout, "metrics: ", log.Lmicroseconds))
	}()

	go func() {
		time.Sleep(10 * time.Second)
		cancelFunc()
	}()
	<-overChannel
}

func makeRestyCall(restyClient *resty.Client, counter dwMetric.Counter, errCounter dwMetric.Counter) {
	res, err := restyClient.R().Get("/delay/10_ms")
	if err == nil {
		counter.Inc(1)
	} else {
		errCounter.Inc(1)
	}
	_ = res
}
