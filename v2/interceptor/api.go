package interceptor

import (
	"context"
)

type Config struct {
	Disabled   bool        `json:"disabled" yaml:"disabled"`
	HmacConfig *HmacConfig `json:"hmac_config" yaml:"hmac_config"`
}

type HmacConfig struct {
	Disabled      bool   `json:"disabled" yaml:"disabled"`
	Key           string `json:"key" yaml:"key"`
	HashHeaderKey string `json:"hash_header_key" yaml:"hash_header_key"`
}

type Interceptor interface {
	Info() (name string, enabled bool)
	Intercept(ctx context.Context, body any) (bodyModified bool, modifiedBody any, err error)
	EnrichHeaders(ctx context.Context) (headers map[string]string, err error)
}

func NewInterceptor(config *Config) Interceptor {

	// No-Op if config is missing
	if config == nil {
		return &noOpInterceptor{}
	}

	// Make correct interceptor based on config
	if config.HmacConfig != nil {
		return &hmacSha256Interceptor{config: config.HmacConfig}
	}

	// Default to no-op implementation
	return &noOpInterceptor{}
}
