package goxHttpApi

import (
	"context"
	"github.com/devlibx/gox-base"
	"github.com/devlibx/gox-base/errors"
	"github.com/devlibx/gox-base/serialization"
	"github.com/devlibx/gox-http/v2/command"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
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
    timeout: 1000
    acceptable_codes: 200,201,204
`

type helperTestSuite struct {
	suite.Suite
	R          *gin.Engine
	testServer *httptest.Server
	goxHttpCtx GoxHttpContext
}

func (s *helperTestSuite) SetupSuite() {
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
