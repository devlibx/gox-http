package goxHttpApi

import (
	"context"
	"fmt"
	"github.com/devlibx/gox-base"
	"github.com/devlibx/gox-base/errors"
	"github.com/devlibx/gox-http/command"
	httpCommand "github.com/devlibx/gox-http/command/http"
	"go.uber.org/zap"
	"net/http"
	"sync"
	"time"
)

// Implementation of http context
type goxHttpContextImpl struct {
	gox.CrossFunction
	logger   *zap.Logger
	config   *command.Config
	commands map[string]command.Command
	timeouts map[string]int
	lock     *sync.Mutex
}

func (g *goxHttpContextImpl) Execute(ctx context.Context, api string, request *command.GoxRequest) (*command.GoxResponse, error) {
	if cmd, ok := g.commands[api]; !ok {
		return nil, &command.GoxHttpError{
			Err:        ErrCommandNotRegisteredForApi,
			StatusCode: http.StatusBadRequest,
			Message:    fmt.Sprintf("command to execute not found: name=%s", api),
			ErrorCode:  "command_not_found",
			Body:       nil,
		}
	} else {

		// Setup context with timeout
		timeout := g.timeouts[api]
		newCtx, ctxCancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
		defer ctxCancel()

		return cmd.Execute(newCtx, request)
	}
}

func (g *goxHttpContextImpl) ExecuteAsync(ctx context.Context, api string, request *command.GoxRequest) chan *command.GoxResponse {
	panic("implement me")
}

// Internal setup method
func (g *goxHttpContextImpl) setup() error {
	g.config.SetupDefaults()

	// Setup timeouts
	g.timeouts = map[string]int{}

	for apiName, api := range g.config.Apis {

		// Find the server used in this API
		server, err := g.config.FindServerByName(api.Server)
		if err != nil {
			return errors.Wrap(err, "failed to create http command (server not found): api=%s", apiName)
		}

		// Create http command for this API
		var cmd command.Command
		if api.DisableHystrix {
			cmd, err = httpCommand.NewHttpCommand(g.CrossFunction, server, api)
		} else {
			cmd, err = httpCommand.NewHttpHystrixCommand(g.CrossFunction, server, api)
		}
		if err != nil {
			return errors.Wrap(err, "failed to create http command: api=%s", apiName)
		}

		// Store this http command to use
		g.commands[apiName] = cmd
		// g.timeouts[apiName] = api.Timeout
		g.timeouts[apiName] = api.GetTimeoutWithRetryIncluded()

	}
	return nil
}

func (g *goxHttpContextImpl) ReloadApi(apiToReload string) error {

	// Lock for updating new resources
	g.lock.Lock()
	defer g.lock.Unlock()

	// Setup defaults
	g.config.SetupDefaults()

	// Update or add existing API
	var err error
	for apiName, api := range g.config.Apis {
		if apiName == apiToReload {
			if _, ok := g.commands[apiName]; ok {
				err = g.updateAPi(api)
			} else {
				err = g.addNewAPi(api)
			}
		}
	}
	return err
}

func (g *goxHttpContextImpl) addNewAPi(api *command.Api) error {
	apiName := api.Name

	// Find the server used in this API
	server, err := g.config.FindServerByName(api.Server)
	if err != nil {
		return errors.Wrap(err, "failed to create http command (server not found): api=%s", apiName)
	}

	// Create http command for this API
	var cmd command.Command
	if api.DisableHystrix {
		cmd, err = httpCommand.NewHttpCommand(g.CrossFunction, server, api)
	} else {
		cmd, err = httpCommand.NewHttpHystrixCommand(g.CrossFunction, server, api)
	}
	if err != nil {
		return errors.Wrap(err, "failed to create http command: api=%s", apiName)
	}

	// Store this http command to use
	g.commands[apiName] = cmd
	g.timeouts[apiName] = api.Timeout

	return nil
}

func (g *goxHttpContextImpl) updateAPi(api *command.Api) error {
	apiName := api.Name

	// Find the server used in this API
	server, err := g.config.FindServerByName(api.Server)
	if err != nil {
		return errors.Wrap(err, "failed to create http command (server not found): api=%s", apiName)
	}

	var updatedCommand command.Command
	if _, ok := g.commands[apiName].(*httpCommand.HttpCommand); ok {
		updatedCommand, err = httpCommand.NewHttpCommand(g.CrossFunction, server, api)
	} else if _cmd, ok := g.commands[apiName].(*httpCommand.HttpHystrixCommand); ok {
		var cmd command.Command
		cmd, err = httpCommand.NewHttpCommand(g.CrossFunction, server, api)
		if err == nil {
			_cmd.UpdateCommand(cmd)
			updatedCommand = _cmd
		}
	}

	if err != nil {
		return errors.Wrap(err, "failed to create http command: api=%s", apiName)
	}

	// Store this http command to use
	g.commands[apiName] = updatedCommand
	g.timeouts[apiName] = api.Timeout

	return nil
}
