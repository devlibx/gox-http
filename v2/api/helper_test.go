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
			c.String(http.StatusOK, `{"id":1, "slug":"lorem-ipsum", "url":"https://jsonplaceholder.org/posts/lorem-ipsum", "title":"Lorem ipsum dolor sit amet, consectetur adipiscing elit.", "content":"Ante taciti nulla sit libero orci sed nam. Sagittis suspendisse gravida ornare iaculis cras nullam varius ac ullamcorper. Nunc euismod hendrerit netus ligula aptent potenti. Aliquam volutpat nibh scelerisque at. Ipsum molestie phasellus euismod sagittis mauris, erat ut. Gravida morbi, sagittis blandit quis ipsum mi mus semper dictum amet himenaeos. Accumsan non congue praesent interdum habitasse turpis orci. Ante curabitur porttitor ullamcorper sagittis sem donec, inceptos cubilia venenatis ac. Augue fringilla sodales in ullamcorper enim curae; rutrum hac in sociis! Scelerisque integer varius et euismod aenean nulla. Quam habitasse risus nullam enim. Ultrices etiam viverra mattis aliquam? Consectetur velit vel volutpat eget curae;. Volutpat class mus elementum pulvinar! Nisi tincidunt volutpat consectetur. Primis morbi pulvinar est montes diam himenaeos duis elit est orci. Taciti sociis aptent venenatis dui malesuada dui justo faucibus primis consequat volutpat. Rhoncus ante purus eros nibh, id et hendrerit pellentesque scelerisque vehicula sollicitudin quam. Hac class vitae natoque tortor dolor dui praesent suspendisse. Vehicula euismod tincidunt odio platea aenean habitasse neque ad proin. Bibendum phasellus enim fames risus eget felis et sem fringilla etiam. Integer.","image":"https://dummyimage.com/800x430/FFFFFF/lorem-ipsum.png&text=jsonplaceholder.org", "thumbnail":"https://dummyimage.com/200x200/FFFFFF/lorem-ipsum.png&text=jsonplaceholder.org", "status":"published", "category":"lorem", "publishedAt":"04/02/2023 13:25:21","updatedAt":"14/03/2023 17:22:20", "userId":1}`)
		} else if id == "GOOD_NO_CONTENT" {
			c.Status(http.StatusNoContent)
		} else if id == "NOT_FOUND" {
			c.String(http.StatusNotFound, `{"status": "some error"}`)
		} else if id == "ERROR" {
			c.Status(http.StatusInternalServerError)
		} else if id == "GOOD_LIST" {
			c.String(http.StatusOK, `[{"id": 1, "name":"a"}, {"id": 2, "name":"b"}]`)
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
	assert.Equal(s.T(), "lorem-ipsum", successResponse.Response.Slug)
}

func (s *helperTestSuite) TestExecuteHttp_Good_List() {

	successResponse, err := ExecuteHttpListResponse[successListPojo, errorPojo](
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
	assert.Equal(s.T(), "a", successResponse.Response[0].Name)
	assert.Equal(s.T(), 2, successResponse.Response[1].Id)
	assert.Equal(s.T(), "b", successResponse.Response[1].Name)
}

func (s *helperTestSuite) TestExecuteHttp_Good_Empty_List() {

	successResponse, err := ExecuteHttpListResponse[successListPojo, errorPojo](
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

type successListPojo struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}
type successPojo struct {
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

type errorPojo struct {
	Status string `json:"status"`
}
