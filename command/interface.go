package command

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/devlibx/gox-base/v2"
	"github.com/devlibx/gox-base/v2/serialization"
	"github.com/devlibx/gox-http/v4/interceptor"
)

//go:generate mockgen -source=interface.go -destination=../mocks/command/mock_interface.go -package=mockGoxHttp

// List of all servers
type Servers map[string]*Server

// Defines a single server
// ****************************************************************************************
// IMP NOTE - "config_parser.go -> UnmarshalYAML() method is created to do custom parsing.
// If you change anything here (add/update/delete) you must make changes in UnmarshalYAML()
// ****************************************************************************************
type Server struct {
	Name                        string
	ProxyUrl                    string                 `yaml:"proxy_url"`
	Host                        string                 `yaml:"host"`
	Port                        int                    `yaml:"port"`
	Https                       bool                   `yaml:"https"`
	SkipCertVerify              string                 `yaml:"skip_cert_verify"`
	ConnectTimeout              int                    `yaml:"connect_timeout"`
	ConnectionRequestTimeout    int                    `yaml:"connection_request_timeout"`
	Properties                  map[string]interface{} `yaml:"properties"`
	Headers                     map[string]interface{} `yaml:"headers"`
	InterceptorConfig           *interceptor.Config    `yaml:"interceptor_config"`
	EnableHttpConnectionTracing bool                   `yaml:"enable_http_connection_tracing"`
}

// List of all APIs
type Apis map[string]*Api

// A single API
// ****************************************************************************************
// IMP NOTE - "config_parser.go -> UnmarshalYAML() method is created to do custom parsing.
// If you change anything here (add/update/delete) you must make changes in UnmarshalYAML()
// ****************************************************************************************
type Api struct {
	Name                         string
	Method                       string              `yaml:"method"`
	Path                         string              `yaml:"path"`
	Server                       string              `yaml:"server"`
	Timeout                      int                 `yaml:"timeout"`
	Concurrency                  int                 `yaml:"concurrency"`
	QueueSize                    int                 `yaml:"queue_size"`
	Async                        bool                `yaml:"async"`
	AcceptableCodes              string              `yaml:"acceptable_codes"`
	RetryCount                   int                 `yaml:"retry_count"`
	InitialRetryWaitTimeMs       int                 `yaml:"retry_initial_wait_time_ms"`
	Headers                      map[string]string   `yaml:"headers"`
	InterceptorConfig            *interceptor.Config `yaml:"interceptor_config"`
	EnableRequestResponseLogging bool                `yaml:"enable_request_response_logging"`
	EnableHttpConnectionTracing  bool                `yaml:"enable_http_connection_tracing"`
	DisableHystrix               bool                `yaml:"disable_hystrix"`
	acceptableCodes              []int
}

func (a *Api) GetTimeoutWithRetryIncluded() int {

	if a.RetryCount <= 0 {
		return a.Timeout
	}

	// Set timeout + 10% delta
	timeout := a.Timeout

	// Add extra time to handle retry counts
	if a.RetryCount > 0 {
		timeout = timeout + (timeout * a.RetryCount) + a.InitialRetryWaitTimeMs
	}

	if timeout/10 <= 0 {
		timeout += 2
	} else {
		timeout += timeout / 10
	}

	return timeout
}

// ****************************************************************************************
// IMP NOTE - "config_parser.go -> UnmarshalYAML() method is created to do custom parsing.
// If you change anything here (add/update/delete) you must make changes in UnmarshalYAML()
// ****************************************************************************************
type Config struct {
	Env     string  `yaml:"env"`
	Servers Servers `yaml:"servers"`
	Apis    Apis    `yaml:"apis"`
}

// ------------------------------------------------------ Request/Response ---------------------------------------------

type MultivaluedMap map[string][]string

type BodyProvider interface {
	Body(object interface{}) ([]byte, error)
}

type ResponseBuilder interface {
	Response(data []byte) (interface{}, error)
}

type GoxRequest struct {
	Api             string          `json:"-"`
	Header          http.Header     `json:"-"`
	PathParam       MultivaluedMap  `json:"path_param,omitempty"`
	QueryParam      MultivaluedMap  `json:"query_param,omitempty"`
	Body            interface{}     `json:"body"`
	BodyProvider    BodyProvider    `json:"-"`
	ResponseBuilder ResponseBuilder `json:"-"`
}

type GoxResponse struct {
	Body       []byte
	Response   interface{}
	StatusCode int
	Err        error
}

func (r *GoxResponse) AsStringObjectMapOrEmpty() gox.StringObjectMap {
	if d, ok := r.Response.(*gox.StringObjectMap); ok {
		return *d
	} else if r.Body != nil {
		if d, err := gox.StringObjectMapFromString(string(r.Body)); err == nil {
			return d
		} else {
			return gox.StringObjectMap{}
		}
	}
	return nil
}

func (r *GoxResponse) String() string {
	if r.Err != nil {
		return fmt.Sprintf("SatusCode=%d, Err=%v", r.StatusCode, r.Err)
	} else if r.Response != nil {
		return fmt.Sprintf("SatusCode=%d, Response=%v", r.StatusCode, r.Response)
	} else if r.Body != nil {
		return fmt.Sprintf("SatusCode=%d, Body=%v", r.StatusCode, string(r.Body))
	} else {
		return fmt.Sprintf("SatusCode=%d", r.StatusCode)
	}
}

type Command interface {
	Execute(ctx context.Context, request *GoxRequest) (*GoxResponse, error)
}

func (req *GoxRequest) String() string {
	return serialization.StringifySuppressError(req, "{}")
}

// UpdateServerWithUrl updates a server's configuration using a URL string. This method can be used when you have a URL
// which you want to use for a server. It parses the URL and updates the server's host, port, and HTTPS settings.
//
// IMPORTANT: This method should only be used before the GoxHttpContext is initialized. Any changes made after
// GoxHttpContext setup will not take effect. This method is primarily used in test scenarios where you need
// to configure a server to point to a test HTTP server (e.g., httptest.Server).
//
// Example test usage:
//
//	server := httptest.NewServer(handler)
//	defer server.Close()
//	config.UpdateServerWithUrl("testServer", server.URL)
//
// Parameters:
//   - serverName: The name of the server in the Config.Servers map to update
//   - urlStr: A URL string in the format "scheme://host[:port][/path]" (e.g., "http://localhost:8080", "https://api.example.com:443")
//
// The method will:
//   - Set the Host field to the hostname/IP from the URL (without port)
//   - Set the Port field if a port is specified in the URL
//   - Set the Https field to true if the URL scheme is "https", false otherwise
//
// If the URL is invalid or missing a scheme, the server configuration will remain unchanged.
// If the server name doesn't exist in the configuration, no action is taken.
func (c *Config) UpdateServerWithUrl(serverName, urlStr string) {
	if server, ok := c.Servers[serverName]; ok {
		u, err := url.Parse(urlStr)
		if err == nil && u.Scheme != "" {
			// Update host without the port
			server.Host = u.Hostname()

			// Update port if present
			if port, err := strconv.Atoi(u.Port()); err == nil {
				server.Port = port
			}

			// Update https based on scheme
			server.Https = u.Scheme == "https"
		}
	}
}
