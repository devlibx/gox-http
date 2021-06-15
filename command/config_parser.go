package command

import (
	"github.com/devlibx/gox-base"
	"github.com/devlibx/gox-base/config"
	"github.com/devlibx/gox-base/errors"
	"github.com/devlibx/gox-base/util"
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
			var _host = config.ParameterizedValue(valueMap.StringOrDefault("host", "localhost"))
			var _https = config.ParameterizedValue(valueMap.StringOrDefault("https", "false"))
			var _port = config.ParameterizedValue(valueMap.StringOrDefault("port", "80"))
			var _connectTimeout = config.ParameterizedValue(valueMap.StringOrDefault("connect_timeout", "50"))
			var connectionRequestTimeout = config.ParameterizedValue(valueMap.StringOrDefault("connection_request_timeout", "50"))

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
			var path = config.ParameterizedValue(valueMap.StringOrDefault("path", "/"))
			var server = config.ParameterizedValue(valueMap.StringOrEmpty("server"))
			var timeout = config.ParameterizedValue(valueMap.StringOrDefault("timeout", "100"))
			var concurrency = config.ParameterizedValue(valueMap.StringOrDefault("concurrency", "1"))
			var queue_size = config.ParameterizedValue(valueMap.StringOrDefault("queue_size", "10"))
			var async = config.ParameterizedValue(valueMap.StringOrDefault("async", "false"))
			var acceptable_codes = config.ParameterizedValue(valueMap.StringOrDefault("acceptable_codes", "200,201"))
			var retry_count = config.ParameterizedValue(valueMap.StringOrDefault("retry_count", "0"))
			var retry_initial_wait_time_ms = config.ParameterizedValue(valueMap.StringOrDefault("retry_initial_wait_time_ms", "1"))

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
		}
	}

	return nil
}
