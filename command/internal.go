package command

import (
	"fmt"
	"github.com/devlibx/gox-base/errors"
	"github.com/devlibx/gox-base/serialization"
	"github.com/devlibx/gox-base/util"
	"net/http"
	"strconv"
	"strings"
)

func (c *Config) SetupDefaults() {

	// Fill defaults in servers
	if c.Servers != nil {
		for k, v := range c.Servers {
			v.Name = k
			if v.ConnectTimeout <= 0 {
				v.ConnectTimeout = 50
			}
			if v.ConnectionRequestTimeout <= 0 {
				v.ConnectionRequestTimeout = 50
			}
			if v.Port == 0 {
				v.Port = 80
			}
			if util.IsStringEmpty(v.Host) {
				v.Host = "localhost"
			}
		}
	}

	// Fill defaults in apis
	if c.Apis != nil {
		for k, v := range c.Apis {
			v.Name = k
			if v.Timeout <= 0 {
				v.Timeout = 1
			}
			if v.Concurrency <= 0 {
				v.Concurrency = 1
			}
			if v.QueueSize <= 0 {
				v.QueueSize = 1
			}
			if util.IsStringEmpty(v.Method) {
				v.Method = "GET"
			}
			if util.IsStringEmpty(v.AcceptableCodes) {
				v.AcceptableCodes = "200,201"
			}

			v.acceptableCodes = make([]int, 0)
			for _, code := range strings.Split(v.AcceptableCodes, ",") {
				code = strings.TrimSpace(code)
				if i, err := strconv.Atoi(code); err == nil {
					v.acceptableCodes = append(v.acceptableCodes, i)
				}
			}
			if len(v.acceptableCodes) == 0 {
				v.acceptableCodes = append(v.acceptableCodes, 200)
				v.acceptableCodes = append(v.acceptableCodes, 201)
			}

			if v.Headers == nil {
				v.Headers = map[string]string{}
			}
		}
	}
}

func (c *Config) FindServerByName(toFind string) (*Server, error) {
	for name, server := range c.Servers {
		if name == toFind {
			return server, nil
		}
	}
	return nil, errors.New("server not found with %s name", toFind)
}

func (c *Config) FindApiByName(toFind string) (*Api, error) {
	for name, api := range c.Apis {
		if name == toFind {
			return api, nil
		}
	}
	return nil, errors.New("api not found with %s name", toFind)
}

func (a *Api) GetPath(server *Server) string {
	if server.Https {
		return fmt.Sprintf("https://%s:%d%s", server.Host, server.Port, a.Path)
	} else {
		return fmt.Sprintf("http://%s:%d%s", server.Host, server.Port, a.Path)
	}
}

func (a *Api) IsHttpCodeAcceptable(code int) bool {
	for _, c := range a.acceptableCodes {
		if c == code {
			return true
		}
	}
	return false
}

type funcBasedResponseBuilder struct {
	responseBuilderFunc func(data []byte) (interface{}, error)
}

func (f *funcBasedResponseBuilder) Response(data []byte) (interface{}, error) {
	return f.responseBuilderFunc(data)
}

func NewFunctionBasedResponseBuilder(f func(data []byte) (interface{}, error)) ResponseBuilder {
	return &funcBasedResponseBuilder{responseBuilderFunc: f}
}

type jsonToObjectResponseBuilder struct {
	out interface{}
}

func (f *jsonToObjectResponseBuilder) Response(data []byte) (interface{}, error) {
	err := serialization.JsonBytesToObject(data, f.out)
	return f.out, err
}

func NewJsonToObjectResponseBuilder(out interface{}) ResponseBuilder {
	return &jsonToObjectResponseBuilder{out: out}
}

// --------------------------------------- Create Builder --------------------------------------------------------------

type goxRequestBuilder struct {
	request *GoxRequest
	api     string
}

type HeaderValuePair struct {
	Name  string
	Value interface{}
}

func (b *goxRequestBuilder) WithContentTypeJson() *goxRequestBuilder {
	return b.WithHeader("content-type", "application/json")
}

func (b *goxRequestBuilder) WithHeader(name string, value interface{}) *goxRequestBuilder {
	if b.request.Header == nil {
		b.request.Header = http.Header{}
	}
	b.request.Header.Add(name, serialization.StringifySuppressError(value, ""))
	return b
}

func (b *goxRequestBuilder) WithHeaders(headers ...HeaderValuePair) *goxRequestBuilder {
	if b.request.Header == nil {
		b.request.Header = http.Header{}
	}
	for _, hp := range headers {
		b.request.Header.Add(hp.Name, serialization.StringifySuppressError(hp.Value, ""))
	}
	return b
}

func (b *goxRequestBuilder) WithPathParam(name string, value interface{}) *goxRequestBuilder {
	if b.request.PathParam == nil {
		b.request.PathParam = map[string][]string{}
	}
	if _, ok := b.request.PathParam[name]; !ok {
		b.request.PathParam[name] = make([]string, 0)
	}

	if str, ok := value.(string); ok {
		b.request.PathParam[name] = append(b.request.PathParam[name], str)
	} else {
		b.request.PathParam[name] = append(b.request.PathParam[name], serialization.StringifySuppressError(value, ""))
	}
	return b
}

func (b *goxRequestBuilder) WithQueryParam(name string, value interface{}) *goxRequestBuilder {
	if b.request.QueryParam == nil {
		b.request.QueryParam = map[string][]string{}
	}
	if _, ok := b.request.QueryParam[name]; !ok {
		b.request.QueryParam[name] = make([]string, 0)
	}

	if str, ok := value.(string); ok {
		b.request.QueryParam[name] = append(b.request.QueryParam[name], str)
	} else {
		b.request.QueryParam[name] = append(b.request.QueryParam[name], serialization.StringifySuppressError(value, ""))
	}
	return b
}

func (b *goxRequestBuilder) WithResponseBuilder(builder ResponseBuilder) *goxRequestBuilder {
	b.request.ResponseBuilder = builder
	return b
}

func (b *goxRequestBuilder) WithBodyProvider(builder BodyProvider) *goxRequestBuilder {
	b.request.BodyProvider = builder
	return b
}

func (b *goxRequestBuilder) WithBody(body interface{}) *goxRequestBuilder {
	b.request.Body = body
	return b
}

func (b *goxRequestBuilder) Build() *GoxRequest {
	return b.request
}

func NewGoxRequestBuilder(api string) *goxRequestBuilder {
	return &goxRequestBuilder{
		request: &GoxRequest{},
		api:     api,
	}
}
