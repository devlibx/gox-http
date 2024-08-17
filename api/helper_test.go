package goxHttpApi

import (
	"context"
	"github.com/devlibx/gox-base/v2"
	"github.com/devlibx/gox-base/v2/errors"
	"github.com/devlibx/gox-base/v2/serialization"
	"github.com/devlibx/gox-http/v3/command"
	"github.com/gin-gonic/gin"
	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	slogjson "github.com/veqryn/slog-json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"
)

var httpConfig = `
servers: 
  testServer:
    host: localhost
    port: 9123

apis:
  getPosts:
    method: GET
    path: /posts/{id}
    server: testServer
    timeout: 100
    acceptable_codes: 200,201,204 
`

type helperTestSuite struct {
	suite.Suite
	R          *gin.Engine
	testServer *httptest.Server
	goxHttpCtx GoxHttpContext
}

func (s *helperTestSuite) SetupSuite() {
	GoxHttpRequestResponseLoggingEnabled = slog.LevelInfo
	h := slogjson.NewHandler(os.Stdout, &slogjson.HandlerOptions{
		AddSource:   false,
		Level:       GoxHttpRequestResponseLoggingEnabled,
		ReplaceAttr: nil, // Same signature and behavior as stdlib JSONHandler
		JSONOptions: json.JoinOptions(
			// Options from the json v2 library (these are the defaults)
			json.Deterministic(true),
			jsontext.AllowDuplicateNames(true),
			jsontext.AllowInvalidUTF8(true),
			jsontext.EscapeForJS(false),
			jsontext.SpaceAfterColon(false),
			jsontext.SpaceAfterComma(true),
		),
	})
	slog.SetDefault(slog.New(h))

	t := s.T()
	s.R = gin.New()
	s.R.GET("/posts/:id", func(c *gin.Context) {
		id := c.Param("id")
		if id == "GOOD" {
			c.String(http.StatusOK, `{"id":1, "name":"user_1"}`)
		} else if id == "GOOD_NO_CONTENT" {
			c.Status(http.StatusNoContent)
		} else if id == "NOT_FOUND" {
			c.String(http.StatusNotFound, `{"status": "some error"}`)
		} else if id == "ERROR" {
			c.Status(http.StatusInternalServerError)
		} else if id == "GOOD_LIST" {
			c.String(http.StatusOK, `[{"id":1, "name":"user_1"}, {"id":2, "name":"user_2"}]`)
		} else if id == "GOOD_EMPTY_LIST" {
			c.String(http.StatusOK, `[]`)
		} else if id == "NOT_FOUND_WITH_NO_BODY" {
			c.Status(http.StatusNotFound)
		} else if id == "HYSTRIX" {
			time.Sleep(200 * time.Millisecond)
			c.String(http.StatusOK, `{"id":1, "name":"user_1"}`)
		} else {
			c.Status(http.StatusInternalServerError)
		}
	})
	s.testServer = httptest.NewServer(s.R)
	_ = t

	cf := gox.NewCrossFunction()

	// Read config and
	config := command.Config{}
	err := serialization.ReadYamlFromString(httpConfig, &config)
	if err != nil {
		slog.Error("got error in reading config", err)
		return
	}

	parsedURL, err := url.Parse(s.testServer.URL)
	assert.NoError(t, err)
	port, err := strconv.Atoi(parsedURL.Port())
	assert.NoError(t, err)
	config.Servers["testServer"].Port = port

	// Setup goHttp context
	s.goxHttpCtx, err = NewGoxHttpContext(cf, &config)
	assert.NoError(t, err)
}

func (s *helperTestSuite) TestExecuteHttp_Good() {
	request := command.NewGoxRequestBuilder("getPosts").
		WithContentTypeJson().
		WithPathParam("id", "GOOD").
		Build()
	successResponse, err := ExecuteHttp[successPojo, errorPojo](context.Background(), s.goxHttpCtx, request)

	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), successResponse)
	assert.NotNil(s.T(), successResponse.Response)
	assert.NotNil(s.T(), successResponse.Body)
	assert.Equal(s.T(), http.StatusOK, successResponse.StatusCode)
	assert.Equal(s.T(), "user_1", successResponse.Response.Name)
}

func (s *helperTestSuite) TestExecuteHttp_Good_List() {

	successResponse, err := ExecuteHttpListResponse[successPojo, errorPojo](
		context.Background(),
		s.goxHttpCtx,
		command.NewGoxRequestBuilder("getPosts").
			WithContentTypeJson().
			WithPathParam("id", "GOOD_LIST").
			Build())

	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), successResponse)
	assert.NotNil(s.T(), successResponse.Response)
	assert.NotNil(s.T(), successResponse.Body)
	assert.Equal(s.T(), http.StatusOK, successResponse.StatusCode)

	assert.True(s.T(), len(successResponse.Response) == 2)
	assert.Equal(s.T(), 1, successResponse.Response[0].Id)
	assert.Equal(s.T(), "user_1", successResponse.Response[0].Name)
	assert.Equal(s.T(), 2, successResponse.Response[1].Id)
	assert.Equal(s.T(), "user_2", successResponse.Response[1].Name)
}

func (s *helperTestSuite) TestExecuteHttp_Good_Empty_List() {

	successResponse, err := ExecuteHttpListResponse[successPojo, errorPojo](
		context.Background(),
		s.goxHttpCtx,
		command.NewGoxRequestBuilder("getPosts").
			WithContentTypeJson().
			WithPathParam("id", "GOOD_EMPTY_LIST").
			Build())

	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), successResponse)
	assert.NotNil(s.T(), successResponse.Response)
	assert.NotNil(s.T(), successResponse.Body)
	assert.Equal(s.T(), http.StatusOK, successResponse.StatusCode)
	assert.True(s.T(), len(successResponse.Response) == 0)
}

func (s *helperTestSuite) TestExecuteHttp_Good_NoContent() {
	request := command.NewGoxRequestBuilder("getPosts").
		WithContentTypeJson().
		WithPathParam("id", "GOOD_NO_CONTENT").
		Build()
	successResponse, err := ExecuteHttp[any, errorPojo](context.Background(), s.goxHttpCtx, request)

	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), successResponse)
	assert.Nil(s.T(), successResponse.Response)
	assert.Nil(s.T(), successResponse.Body)
	assert.Equal(s.T(), http.StatusNoContent, successResponse.StatusCode)
}

func (s *helperTestSuite) TestExecuteHttp_Bad_NotFound() {
	request := command.NewGoxRequestBuilder("getPosts").
		WithContentTypeJson().
		WithPathParam("id", "NOT_FOUND").
		Build()
	successResponse, err := ExecuteHttp[successPojo, errorPojo](context.Background(), s.goxHttpCtx, request)

	assert.Error(s.T(), err)
	assert.Nil(s.T(), successResponse)
	errorResponse, errorResponsePayloadExists, ok := ExtractError[errorPojo](err)
	assert.True(s.T(), ok)
	assert.NotNil(s.T(), errorResponse)
	assert.True(s.T(), errorResponsePayloadExists)
	assert.Equal(s.T(), "some error", errorResponse.Response.Status)
	assert.Equal(s.T(), http.StatusNotFound, errorResponse.StatusCode)
}

// Test case to check if we can get error without error pojo
// We will get the error as map[string]interface{}
func (s *helperTestSuite) TestExecuteHttp_Bad_NotFound_With_NoErrorPojo() {
	request := command.NewGoxRequestBuilder("getPosts").
		WithContentTypeJson().
		WithPathParam("id", "NOT_FOUND").
		Build()
	successResponse, err := ExecuteHttp[successPojo, any](context.Background(), s.goxHttpCtx, request)

	assert.Error(s.T(), err)
	assert.Nil(s.T(), successResponse)
	errorResponse, errorResponsePayloadExists, ok := ExtractError[any](err)
	assert.True(s.T(), ok)
	assert.NotNil(s.T(), errorResponse)
	assert.True(s.T(), errorResponsePayloadExists)
	if asMap, ok := errorResponse.Response.(map[string]interface{}); ok {
		assert.Equal(s.T(), "some error", asMap["status"])
	} else {
		assert.Fail(s.T(), "we should have got it as map")
	}
	assert.Equal(s.T(), http.StatusNotFound, errorResponse.StatusCode)
}

// Test case to check if we can get error without error pojo
// We will get the error as map[string]interface{}
func (s *helperTestSuite) TestExecuteHttp_Bad_NotFound_With_NoErrorPojo_AndNoBodyFromApi() {
	request := command.NewGoxRequestBuilder("getPosts").
		WithContentTypeJson().
		WithPathParam("id", "NOT_FOUND_WITH_NO_BODY").
		Build()
	successResponse, err := ExecuteHttp[successPojo, any](context.Background(), s.goxHttpCtx, request)

	assert.Error(s.T(), err)
	assert.Nil(s.T(), successResponse)
	errorResponse, errorResponsePayloadExists, ok := ExtractError[any](err)
	assert.True(s.T(), ok)
	assert.NotNil(s.T(), errorResponse)
	assert.False(s.T(), errorResponsePayloadExists)
	asMap, ok := errorResponse.Response.(map[string]interface{})
	assert.False(s.T(), ok)
	assert.Nil(s.T(), asMap)

	assert.Equal(s.T(), http.StatusNotFound, errorResponse.StatusCode)
}

func (s *helperTestSuite) TestExecuteHttp_Bad_Error() {
	successResponse, err := ExecuteHttp[successPojo, errorPojo](
		context.Background(),
		s.goxHttpCtx,
		command.NewGoxRequestBuilder("getPosts").
			WithContentTypeJson().
			WithPathParam("id", "ERROR").
			Build(),
	)

	assert.Error(s.T(), err)
	assert.Nil(s.T(), successResponse)
	errorResponse, hasResponse, ok := ExtractError[errorPojo](err)
	assert.True(s.T(), ok)
	assert.NotNil(s.T(), errorResponse)
	assert.Equal(s.T(), http.StatusInternalServerError, errorResponse.StatusCode)
	assert.False(s.T(), hasResponse)

	var goxError *command.GoxHttpError
	assert.True(s.T(), errors.As(err, &goxError))
	assert.Equal(s.T(), http.StatusInternalServerError, goxError.StatusCode)
}

func (s *helperTestSuite) TestExecuteHttp_Bad_Hystrix() {
	request := command.NewGoxRequestBuilder("getPosts").
		WithContentTypeJson().
		WithPathParam("id", "HYSTRIX").
		Build()
	_, err := ExecuteHttp[successPojo, errorPojo](context.Background(), s.goxHttpCtx, request)
	assert.Error(s.T(), err)

	var goxError *command.GoxHttpError
	assert.True(s.T(), errors.As(err, &goxError))
	assert.True(s.T(), goxError.IsRequestTimeout())
}

func (s *helperTestSuite) TearDownSuite() {
	t := s.T()
	_ = t
}

func TestHttpHelperSuite(t *testing.T) {
	suite.Run(t, new(helperTestSuite))
}

type successPojo struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type errorPojo struct {
	Status string `json:"status"`
}
