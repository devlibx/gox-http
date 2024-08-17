package interceptor

import (
	"context"
	"github.com/devlibx/gox-base/v2/errors"
	"github.com/devlibx/gox-base/v2/serialization"
)

type Config struct {
	Disabled   bool        `json:"disabled" yaml:"disabled"`
	HmacConfig *HmacConfig `json:"hmac_config" yaml:"hmac_config"`
}

func (cfg *Config) PopulateFromMap(input map[string]interface{}, debugString string) error {
	if out, err := serialization.Stringify(input); err != nil {
		return errors.Wrap(err, "error is stringfy interceptor_config property: info=%s", debugString)
	} else {
		if err := serialization.JsonBytesToObject([]byte(out), &cfg); err != nil {
			return errors.Wrap(err, "error is parsing interceptor_config property: info=%s", debugString)
		}
	}
	return nil
}

type HmacConfig struct {
	Disabled bool   `json:"disabled" yaml:"disabled"`
	Key      string `json:"key" yaml:"key"`

	DumpDebug bool `json:"dump_debug" yaml:"dump_debug"`

	HashHeaderKey      string `json:"hash_header_key" yaml:"hash_header_key"`
	TimestampHeaderKey string `json:"timestamp_header_key" yaml:"timestamp_header_key"`

	// Headers to include in signature - what all headers you should include in signature
	HeadersToIncludeInSignature  []string `json:"headers_to_include_in_signature" yaml:"headers_to_include_in_signature"`
	ConvertHeaderKeysToLowerCase bool     `json:"convert_header_keys_to_lower_case" yaml:"convert_header_keys_to_lower_case"`
}

type Interceptor interface {
	Info() (name string, enabled bool)
	Intercept(ctx context.Context, body any) (bodyModified bool, modifiedBody any, err error)
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
