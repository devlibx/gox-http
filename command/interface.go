package command

import (
	"context"
	"fmt"
	"github.com/devlibx/gox-base"
	"net/http"
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
	Name                     string
	Host                     string `yaml:"host"`
	Port                     int    `yaml:"port"`
	Https                    bool   `yaml:"https"`
	ConnectTimeout           int    `yaml:"connect_timeout"`
	ConnectionRequestTimeout int    `yaml:"connection_request_timeout"`
}

// List of all APIs
type Apis map[string]*Api

// A single API
// ****************************************************************************************
// IMP NOTE - "config_parser.go -> UnmarshalYAML() method is created to do custom parsing.
// If you change anything here (add/update/delete) you must make changes in UnmarshalYAML()
// ****************************************************************************************
type Api struct {
	Name                   string
	Method                 string            `yaml:"method"`
	Path                   string            `yaml:"path"`
	Server                 string            `yaml:"server"`
	Timeout                int               `yaml:"timeout"`
	Concurrency            int               `yaml:"concurrency"`
	QueueSize              int               `yaml:"queue_size"`
	Async                  bool              `yaml:"async"`
	AcceptableCodes        string            `yaml:"acceptable_codes"`
	RetryCount             int               `yaml:"retry_count"`
	InitialRetryWaitTimeMs int               `yaml:"retry_initial_wait_time_ms"`
	Headers                map[string]string `yaml:"headers"`
	acceptableCodes        []int
	DisableHystrix         bool
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
	Header          http.Header
	PathParam       MultivaluedMap
	QueryParam      MultivaluedMap
	Body            interface{}
	BodyProvider    BodyProvider
	ResponseBuilder ResponseBuilder
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
	ExecuteAsync(ctx context.Context, request *GoxRequest) chan *GoxResponse
}

func (req *GoxRequest) String() string {
	return fmt.Sprintf("")
}
