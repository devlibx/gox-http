package command

import (
	"github.com/devlibx/gox-base"
	"github.com/devlibx/gox-base/errors"
	"github.com/devlibx/gox-base/serialization"
	"github.com/devlibx/gox-base/util"
	"github.com/devlibx/gox-http/v2/interceptor"
)

func (e *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	data := map[string]interface{}{}
	if err := unmarshal(&data); err != nil {
		return err
	}

	var sm gox.StringObjectMap = data
	e.Env = sm.StringOrDefault("env", "prod")
	if util.IsStringEmpty(e.Env) {
		e.Env = "prod"
	}
	e.Servers = map[string]*Server{}
	e.Apis = map[string]*Api{}

	if servers, ok := sm["servers"].(map[string]interface{}); ok {
		for name, values := range servers {
			if _, ok := values.(map[string]interface{}); !ok {
				return errors.New("expected data to be type of map for server=%s", name)
			}

			var err error
			s := &Server{Name: name}
			e.Servers[name] = s

			var valueMap gox.StringObjectMap = values.(map[string]interface{})
			var _host = serialization.ParameterizedValue(valueMap.StringOrDefault("host", "localhost"))
			var _https = serialization.ParameterizedValue(valueMap.StringOrDefault("https", "false"))
			var _port = serialization.ParameterizedValue(valueMap.StringOrDefault("port", "80"))
			var _connectTimeout = serialization.ParameterizedValue(valueMap.StringOrDefault("connect_timeout", "50"))
			var connectionRequestTimeout = serialization.ParameterizedValue(valueMap.StringOrDefault("connection_request_timeout", "50"))

			if s.Host, err = _host.GetString(e.Env); err != nil {
				return errors.Wrap(err, "error is parsing host property for server=%s", name)
			}
			if s.Port, err = _port.GetInt(e.Env); err != nil {
				return errors.Wrap(err, "error is parsing port property for server=%s", name)
			}
			if s.Https, err = _https.GetBool(e.Env); err != nil {
				return errors.Wrap(err, "error is parsing https property for server=%s", name)
			}
			if s.ConnectTimeout, err = _connectTimeout.GetInt(e.Env); err != nil {
				return errors.Wrap(err, "error is parsing connect_timeout property for server=%s", name)
			}
			if s.ConnectionRequestTimeout, err = connectionRequestTimeout.GetInt(e.Env); err != nil {
				return errors.Wrap(err, "error is parsing connection_request_timeout property for server=%s", name)
			}
			if m, ok := valueMap["properties"]; ok {
				if _m, ok := m.(map[string]interface{}); ok {
					s.Properties = _m
				}
			}
			if m, ok := valueMap["headers"]; ok {
				if _m, ok := m.(map[string]interface{}); ok {
					s.Headers = _m
				}
			}
			if m, ok := valueMap["interceptor_config"].(map[string]interface{}); ok {
				s.InterceptorConfig = &interceptor.Config{}
				if err := s.InterceptorConfig.PopulateFromMap(m, "server="+name); err != nil {
					return err
				}
			}
		}
	}

	if servers, ok := sm["apis"].(map[string]interface{}); ok {
		for name, values := range servers {
			if _, ok := values.(map[string]interface{}); !ok {
				return errors.New("expected data to be type of map for api=%s", name)
			}

			var err error
			a := &Api{Name: name}
			e.Apis[name] = a

			var valueMap gox.StringObjectMap = values.(map[string]interface{})
			a.Method = valueMap.StringOrDefault("method", "GET")
			var path = serialization.ParameterizedValue(valueMap.StringOrDefault("path", "/"))
			var server = serialization.ParameterizedValue(valueMap.StringOrEmpty("server"))
			var timeout = serialization.ParameterizedValue(valueMap.StringOrDefault("timeout", "100"))
			var concurrency = serialization.ParameterizedValue(valueMap.StringOrDefault("concurrency", "1"))
			var queue_size = serialization.ParameterizedValue(valueMap.StringOrDefault("queue_size", "10"))
			var async = serialization.ParameterizedValue(valueMap.StringOrDefault("async", "false"))
			var acceptable_codes = serialization.ParameterizedValue(valueMap.StringOrDefault("acceptable_codes", "200,201"))
			var retry_count = serialization.ParameterizedValue(valueMap.StringOrDefault("retry_count", "0"))
			var retry_initial_wait_time_ms = serialization.ParameterizedValue(valueMap.StringOrDefault("retry_initial_wait_time_ms", "1"))
			var enable_request_response = serialization.ParameterizedValue(valueMap.StringOrDefault("enable_request_response_logging", "false"))

			if a.Path, err = path.GetString(e.Env); err != nil {
				return errors.Wrap(err, "error is parsing path property for api=%s", name)
			}
			if a.Server, err = server.GetString(e.Env); err != nil {
				return errors.Wrap(err, "error is parsing server property for api=%s", name)
			}
			if a.Timeout, err = timeout.GetInt(e.Env); err != nil {
				return errors.Wrap(err, "error is parsing timeout property for api=%s", name)
			}
			if a.Concurrency, err = concurrency.GetInt(e.Env); err != nil {
				return errors.Wrap(err, "error is parsing concurrency property for api=%s", name)
			}
			if a.QueueSize, err = queue_size.GetInt(e.Env); err != nil {
				return errors.Wrap(err, "error is parsing queue_size property for api=%s", name)
			}
			if a.Async, err = async.GetBool(e.Env); err != nil {
				return errors.Wrap(err, "error is parsing async property for api=%s", name)
			}
			if a.AcceptableCodes, err = acceptable_codes.GetString(e.Env); err != nil {
				return errors.Wrap(err, "error is parsing acceptable_codes property for api=%s", name)
			}
			if a.RetryCount, err = retry_count.GetInt(e.Env); err != nil {
				return errors.Wrap(err, "error is parsing retry_count property for api=%s", name)
			}
			if a.InitialRetryWaitTimeMs, err = retry_initial_wait_time_ms.GetInt(e.Env); err != nil {
				return errors.Wrap(err, "error is parsing retry_initial_wait_time_ms property for api=%s", name)
			}
			if m, ok := valueMap["interceptor_config"].(map[string]interface{}); ok {
				a.InterceptorConfig = &interceptor.Config{}
				if err := a.InterceptorConfig.PopulateFromMap(m, "api="+name); err != nil {
					return err
				}
			}
			if a.EnableRequestResponseLogging, err = enable_request_response.GetBool(e.Env); err != nil {
				return errors.Wrap(err, "error is parsing enable_request_response_logging property for api=%s", name)
			}
		}
	}

	return nil
}
