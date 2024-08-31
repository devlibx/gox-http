package goxHttpApi

import (
	"context"
	"github.com/devlibx/gox-base"
	"github.com/devlibx/gox-base/errors"
	"github.com/devlibx/gox-http/v2/command"
	"github.com/go-resty/resty/v2"
	"sync"
)

//go:generate mockgen -source=api.go -destination=../mocks/api/mock_api.go -package=mockGoxHttp

var ErrCommandNotRegisteredForApi = errors.New("api not found")

// GoxHttpContext is the interface to be used by external clients
type GoxHttpContext interface {
	ReloadApi(apiToReload string) error
	Execute(ctx context.Context, request *command.GoxRequest) (*command.GoxResponse, error)
}

// RestyClientProvider - Interface to get resty client
type RestyClientProvider interface {
	GetRestyClient(api string) (*resty.Client, bool)
}

// NewGoxHttpContext - Create a new http context to be used
func NewGoxHttpContext(cf gox.CrossFunction, config *command.Config) (GoxHttpContext, error) {
	c := &goxHttpContextImpl{
		CrossFunction: cf,
		logger:        cf.Logger().Named("gox-http"),
		config:        config,
		commands:      map[string]command.Command{},
		lock:          &sync.Mutex{},
	}

	if err := c.setup(); err != nil {
		return nil, err
	}

	return c, nil
}

// GetRestyClientFromGoxHttpCtx - Get resty client from gox http context
//
// Parameters:
// - goHttpCtx - GoxHttpContext to extract resty client which is powering this api
// - apiName - Name of the api
//
// Returns:
// - resty client if this command is implemented using resty client under the hood
// - bool - true if resty client is returned otherwise false
func GetRestyClientFromGoxHttpCtx(goHttpCtx GoxHttpContext, apiName string) (*resty.Client, bool) {
	if restyClientProvider, ok := goHttpCtx.(RestyClientProvider); ok {
		if rc, ok := restyClientProvider.GetRestyClient(apiName); ok {
			return rc, true
		}
	}
	return nil, false
}

// SetupOnBeforeRequestOverRestyClientFromGoxHttpCtx - Setup on before request middleware over resty client
//
// Parameters:
// - goHttpCtx - GoxHttpContext to extract resty client which is powering this api
// - apiName - Name of the api
// - middleware - Middleware to be setup to be called on before request
//
// Returns:
// - bool - true if resty client is returned otherwise false
func SetupOnBeforeRequestOverRestyClientFromGoxHttpCtx(goxHttpCtx GoxHttpContext, apiName string, middleware resty.RequestMiddleware) bool {
	if rc, ok := GetRestyClientFromGoxHttpCtx(goxHttpCtx, apiName); ok {
		rc.OnBeforeRequest(middleware)
		return true
	} else {
		return false
	}
}
