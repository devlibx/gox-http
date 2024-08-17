package goxHttpApi

import (
	"context"
	"github.com/devlibx/gox-base/v2"
	"github.com/devlibx/gox-base/v2/errors"
	"github.com/devlibx/gox-http/v3/command"
	"sync"
)

//go:generate mockgen -source=api.go -destination=../mocks/api/mock_api.go -package=mockGoxHttp

var ErrCommandNotRegisteredForApi = errors.New("api not found")

// GoxHttpContext is the interface to be used by external clients
type GoxHttpContext interface {
	ReloadApi(apiToReload string) error
	Execute(ctx context.Context, request *command.GoxRequest) (*command.GoxResponse, error)
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
