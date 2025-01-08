package goxHttpApi

import (
	"context"
	"github.com/devlibx/gox-base/v2/errors"
	"github.com/devlibx/gox-http/v4/command"
)

type noOpGoxHttpContext struct {
}

func (n noOpGoxHttpContext) ReloadApi(apiToReload string) error {
	return nil
}

func (n noOpGoxHttpContext) Execute(ctx context.Context, request *command.GoxRequest) (*command.GoxResponse, error) {
	return nil, errors.New("not implemented")
}

func NoOpGoxHttpContext() GoxHttpContext {
	return &noOpGoxHttpContext{}
}
