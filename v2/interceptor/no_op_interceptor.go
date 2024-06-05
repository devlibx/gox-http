package interceptor

import "context"

type noOpInterceptor struct {
}

func (n noOpInterceptor) Info() (name string, enabled bool) {
	return "no-op", false
}

func (n noOpInterceptor) Intercept(ctx context.Context, body any) (bodyModified bool, modifiedBody any, err error) {
	return false, body, nil
}

func (n noOpInterceptor) EnrichHeaders(ctx context.Context) (headers map[string]string, err error) {
	return map[string]string{}, nil
}
