package interceptor

import "context"

type noOpInterceptor struct {
}

func (n noOpInterceptor) Info() (name string, enabled bool) {
	return "no-op", false
}

func (n noOpInterceptor) Intercept(ctx context.Context, input any) (bodyModified bool, modifiedInput any, err error) {
	return false, input, nil
}
