package main

import (
	"context"
	"errors"
	"github.com/devlibx/gox-base"
	"github.com/devlibx/gox-base/serialization"
	goxHttpApi "github.com/devlibx/gox-http/v2/api"
	"github.com/devlibx/gox-http/v2/command"
	"log"
	"log/slog"
	"net/http"
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
  getPostsLocal:
    method: GET
    path: /posts/{id}
    server: testServer
    timeout: 1000
    acceptable_codes: 200,201
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

	go func() {
		RunMail()
	}()
	time.Sleep(time.Second)

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

	request := command.NewGoxRequestBuilder("getPostsLocal").
		WithContentTypeJson().
		WithPathParam("id", 2).
		Build()

	_, successResponse, erra := ExecuteHttp[Pojo, PojoError](goxHttpCtx, request)
	if erra != nil {
		slog.Error("[Failed to execute]", slog.Any("status", erra.Response))
		return
	}
	log.Println("Got result", successResponse.Title)
}

type GoxGenericResponse[T any] struct {
	Body       []byte
	Response   T
	StatusCode int
	Err        error
}

func ExecuteHttp[SR any, ER any](goxHttpCtx goxHttpApi.GoxHttpContext, request *command.GoxRequest) (GoxGenericResponse[*SR], *SR, *GoxHttpResponseError[*ER]) {

	resp, err := goxHttpCtx.Execute(context.Background(), request)
	if err != nil {
		var goxError *command.GoxHttpError
		if errors.As(err, &goxError) {
			var er *ER
			if e := serialization.JsonBytesToObject(resp.Body, &er); e == nil {
				return GoxGenericResponse[*SR]{Err: err, Body: resp.Body, StatusCode: resp.StatusCode}, nil, &GoxHttpResponseError[*ER]{Response: er}
			}
		} else {
			return GoxGenericResponse[*SR]{Err: err}, nil, &GoxHttpResponseError[*ER]{Response: nil}
		}
	}

	var r *SR
	if err := serialization.JsonBytesToObject(resp.Body, &r); err != nil {
		return GoxGenericResponse[*SR]{Err: err, Body: resp.Body, StatusCode: resp.StatusCode}, nil, &GoxHttpResponseError[*ER]{Response: nil}
	}

	return GoxGenericResponse[*SR]{Body: resp.Body, StatusCode: resp.StatusCode, Response: r}, r, nil
}

type Pojo struct {
	ID          int    `json:"id"`
	Slug        string `json:"slug"`
	URL         string `json:"url"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	Image       string `json:"image"`
	Thumbnail   string `json:"thumbnail"`
	Status      string `json:"status"`
	Category    string `json:"category"`
	PublishedAt string `json:"publishedAt"`
	UpdatedAt   string `json:"updatedAt"`
	UserID      int    `json:"userId"`
}

type PojoError struct {
	Status string `json:"status"`
}

type GoxHttpResponseError[T any] struct {
	Response T
	P        string
}

func (receiver GoxHttpResponseError[T]) Error() string {
	return ""
}

func (receiver GoxHttpResponseError[T]) Resp() T {
	return receiver.Response
}

func RunMail() {
	http.Handle("/posts/1", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"id":1,"slug":"slug","url":"url","title":"Harish Title","content":"content","image":"image","thumbnail":"thumbnail","status":"status","category":"category","publishedAt":"publishedAt","updatedAt":"updatedAt","userId":1}`))
	}))

	http.Handle("/posts/2", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"status": "error"}`))
	}))

	err := http.ListenAndServe(":9123", nil)
	if err != nil {
		panic(err)
	}
}
