package goxHttpApi

import (
	"context"
	"fmt"
	"github.com/devlibx/gox-base"
	"github.com/devlibx/gox-base/serialization"
	"github.com/devlibx/gox-http/v2/command"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"net/http"
	"net/http/httptest"
	"testing"
)

type RestyProviderTestSuite struct {
	suite.Suite
	goxHttpCtx GoxHttpContext
}

func (s *RestyProviderTestSuite) SetupTest() {
	cf := gox.NewCrossFunction()

	// Read config and
	config := command.Config{}
	err := serialization.ReadYamlFromString(httpConfig, &config)
	assert.NoError(s.T(), err)

	goxHttpCtx, err := NewGoxHttpContext(cf, &config)
	assert.NoError(s.T(), err)
	s.goxHttpCtx = goxHttpCtx
}

func (s *RestyProviderTestSuite) TestRestyProvider() {
	t := s.T()

	// Test goxHttpCtx as a resty client provider
	onBeforeRequestCalled := false
	if rc, ok := GetRestyClientFromGoxHttpCtx(s.goxHttpCtx, "getJsonPlaceholderPosts"); ok {
		rc.OnBeforeRequest(func(client *resty.Client, request *resty.Request) error {
			onBeforeRequestCalled = true
			return nil
		})
	}

	// Make a call to getPosts - this should trigger OnBeforeRequest
	resp, err := s.goxHttpCtx.Execute(context.Background(), command.NewGoxRequestBuilder("getJsonPlaceholderPosts").WithPathParam("id", "1").Build())
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.True(t, onBeforeRequestCalled)
}

func (s *RestyProviderTestSuite) TestRestyProvider_SetupOnBeforeRequestOverRestyClientFromGoxHttpCtx() {
	t := s.T()

	// Test goxHttpCtx as a resty client provider and test helper SetupOnBeforeRequestOverRestyClientFromGoxHttpCtx
	// This will call OnBeforeRequest for the given api
	onBeforeRequestCalled := false
	ok := SetupOnBeforeRequestOverRestyClientFromGoxHttpCtx(
		s.goxHttpCtx,
		"getJsonPlaceholderPosts",
		func(client *resty.Client, request *resty.Request) error {
			onBeforeRequestCalled = true
			return nil
		})
	assert.True(t, ok)

	// Make a call to getPosts - this should trigger OnBeforeRequest
	resp, err := s.goxHttpCtx.Execute(context.Background(), command.NewGoxRequestBuilder("getJsonPlaceholderPosts").WithPathParam("id", "1").Build())
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.True(t, onBeforeRequestCalled)
}

func (s *RestyProviderTestSuite) TestRestyProvider_WithMockServer() {
	t := s.T()

	// Set up a mock server to give response
	httpTestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintln(w, `{"status": "ok"}`)
	}))
	defer httpTestServer.Close()

	// Test goxHttpCtx as a resty client provider
	onBeforeRequestCalled := false
	if rc, ok := GetRestyClientFromGoxHttpCtx(s.goxHttpCtx, "getJsonPlaceholderPosts"); ok {
		rc.OnBeforeRequest(func(client *resty.Client, request *resty.Request) error {
			onBeforeRequestCalled = true
			request.URL = httpTestServer.URL
			return nil
		})
	}

	// Make a call to getPosts - this should trigger OnBeforeRequest
	resp, err := s.goxHttpCtx.Execute(context.Background(), command.NewGoxRequestBuilder("getJsonPlaceholderPosts").WithPathParam("id", "1").Build())
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	respMap := gox.StringObjectMapFromJsonOrEmpty(string(resp.Body))
	assert.Equal(t, "ok", respMap.StringOrEmpty("status"))
	assert.True(t, onBeforeRequestCalled)
}

func (s *RestyProviderTestSuite) TestRestyProvider_WithError() {
	t := s.T()

	// Set up a mock server to give response
	httpTestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintln(w, `{"status": "ok"}`)
	}))
	defer httpTestServer.Close()

	// Test goxHttpCtx as a resty client provider
	onBeforeRequestCalled := false
	if rc, ok := GetRestyClientFromGoxHttpCtx(s.goxHttpCtx, "getJsonPlaceholderPosts"); ok {
		rc.OnBeforeRequest(func(client *resty.Client, request *resty.Request) error {
			onBeforeRequestCalled = true
			request.URL = httpTestServer.URL
			return nil
		})
	}

	// Make a call to getPosts - this should trigger OnBeforeRequest
	resp, err := s.goxHttpCtx.Execute(context.Background(), command.NewGoxRequestBuilder("getJsonPlaceholderPosts").WithPathParam("id", "1").Build())
	assert.Error(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	respMap := gox.StringObjectMapFromJsonOrEmpty(string(resp.Body))
	assert.Equal(t, "ok", respMap.StringOrEmpty("status"))
	assert.True(t, onBeforeRequestCalled)
}

func TestRestyProviderTestSuite(t *testing.T) {
	suite.Run(t, new(RestyProviderTestSuite))
}
