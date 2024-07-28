package httpCommand

import (
	"context"
	"github.com/devlibx/gox-http/v2/command"
	"sync"
)

// EnablePreRequestInterceptor is a global flag to enable pre request interceptor
var EnablePreRequestInterceptor = false

// InterceptorFunc is a function which will be called before making a http request
type InterceptorFunc func(ctx context.Context, request *command.GoxRequest) (bool, *command.GoxResponse, error)

// interceptorFuncMap is a map of interceptor function
var interceptorFuncMap = make(map[string]InterceptorFunc, 0)
var interceptorFuncMapMutex = &sync.Mutex{}

func RegisterPreRequestInterceptorFunc(id string, interceptorFunc InterceptorFunc) {
	interceptorFuncMapMutex.Lock()
	defer interceptorFuncMapMutex.Unlock()
	interceptorFuncMap[id] = interceptorFunc
}

func UnregisterPreRequestInterceptorFunc(id string) {
	interceptorFuncMapMutex.Lock()
	defer interceptorFuncMapMutex.Unlock()
	delete(interceptorFuncMap, id)
}

func (h *HttpCommand) interceptPreRequestInterceptor(ctx context.Context, request *command.GoxRequest) (bool, *command.GoxResponse, error) {
	interceptorFuncMapMutex.Lock()
	defer interceptorFuncMapMutex.Unlock()

	// Go to each interceptor and check if any interceptor is stopping the request
	for _, interceptorFunc := range interceptorFuncMap {
		if stop, resp, err := interceptorFunc(ctx, request); stop {
			return true, resp, err
		}
	}

	// No interceptor found - so return false
	return false, nil, nil
}
