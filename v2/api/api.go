package goxHttpApi

import (
	"context"
	"github.com/devlibx/gox-base"
	"github.com/devlibx/gox-base/errors"
	"github.com/devlibx/gox-http/v2/command"
	"sync"
)

//go:generate mockgen -source=api.go -destination=../mocks/api/mock_api.go -package=mockGoxHttp

var ErrCommandNotRegisteredForApi = errors.New("api not found")

// GoxHttpContext is the interface to be used by external clients
type GoxHttpContext interface {
	ReloadApi(apiToReload string) error
	Execute(ctx context.Context, api string, request *command.GoxRequest) (*command.GoxResponse, error)
	ExecuteAsync(ctx context.Context, api string, request *command.GoxRequest) chan *command.GoxResponse
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
