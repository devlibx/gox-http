package command

import (
	"github.com/devlibx/gox-base/v2/serialization"
	"github.com/devlibx/gox-http/v3/testhelper"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseConfig(t *testing.T) {
	config := Config{}
	err := serialization.ReadYamlFromString(testhelper.TestConfig, &config)
	assert.NoError(t, err)
	assert.True(t, len(config.Servers) > 0)
	assert.True(t, len(config.Apis) > 0)

	assert.Equal(t, "jsonplaceholder.typicode.com", config.Servers["jsonplaceholder"].Host)
	assert.Equal(t, 443, config.Servers["jsonplaceholder"].Port)
	assert.Equal(t, true, config.Servers["jsonplaceholder"].Https)
	assert.Equal(t, 1000, config.Servers["jsonplaceholder"].ConnectTimeout)
	assert.Equal(t, 1000, config.Servers["jsonplaceholder"].ConnectionRequestTimeout)
	assert.Equal(t, "localhost", config.Servers["testServer"].Host)
	assert.Equal(t, 9123, config.Servers["testServer"].Port)

	assert.Equal(t, "GET", config.Apis["getPosts"].Method)
	assert.Equal(t, "/posts/{id}", config.Apis["getPosts"].Path)
	assert.Equal(t, "jsonplaceholder", config.Apis["getPosts"].Server)
	assert.Equal(t, 1000, config.Apis["getPosts"].Timeout)
	assert.Equal(t, "200,201", config.Apis["getPosts"].AcceptableCodes)

	assert.Equal(t, "GET", config.Apis["delay_timeout_5000"].Method)
	assert.Equal(t, "/delay", config.Apis["delay_timeout_5000"].Path)
	assert.Equal(t, "testServer", config.Apis["delay_timeout_5000"].Server)
	assert.Equal(t, 5000, config.Apis["delay_timeout_5000"].Timeout)
	assert.Equal(t, "200,201", config.Apis["delay_timeout_5000"].AcceptableCodes)
	assert.Equal(t, 3, config.Apis["delay_timeout_5000"].Concurrency)
	assert.Equal(t, 10, config.Apis["delay_timeout_5000"].QueueSize)

	assert.Equal(t, "POST", config.Apis["post_api_with_delay_2000"].Method)
	assert.Equal(t, "/delay", config.Apis["post_api_with_delay_2000"].Path)
	assert.Equal(t, "testServer", config.Apis["post_api_with_delay_2000"].Server)
	assert.Equal(t, 2000, config.Apis["post_api_with_delay_2000"].Timeout)
	assert.Equal(t, "200,201", config.Apis["post_api_with_delay_2000"].AcceptableCodes)
	assert.Equal(t, 1, config.Apis["post_api_with_delay_2000"].Concurrency)
	assert.Equal(t, 10, config.Apis["post_api_with_delay_2000"].QueueSize)
}

// servers:
//
//	jsonplaceholder:
//	  host: "env: prod=jsonplaceholder.typicode.com; stage=localhost.stage; default: localhost.dev"
//	  port: "env: prod=443; default=8080"
//	  https: true
//	  connect_timeout: "env: prod=10; default=1000"
//	  connection_request_timeout: "env: prod=11; default=1001"
//	testServer:
//	  host: "env: prod=localhost.prod; dev=localhost.dev; stage=localhost.stage"
//	  port: 9123
//	  https: "env: prod=true; dev=false; stage=false"
//
// apis:
//
//	delay_timeout_10:
//	  path: /delay
//	  server: testServer
//	  timeout: "env: prod=10; default=1000"
//	  concurrency: "env: prod=10; default=300"
//	delay_timeout_10_POST:
//	  path: /delay
//	  method: POST
//	  server: testServer
//	  timeout: "env: prod=100; default=1000"
//	  concurrency: "env: prod=11; default=200"
func TestParseConfigWithParameterizedConfig_WithDefaultEnv(t *testing.T) {
	config := Config{}
	err := serialization.ReadYamlFromString(testhelper.TestConfigWithEnv, &config)
	assert.NoError(t, err)
	assert.True(t, len(config.Servers) > 0)
	assert.True(t, len(config.Apis) > 0)

	serverConfig := config.Servers["jsonplaceholder"]
	assert.Equal(t, "jsonplaceholder.typicode.com", serverConfig.Host)
	assert.Equal(t, 443, serverConfig.Port)
	assert.Equal(t, true, serverConfig.Https)
	assert.Equal(t, 10, serverConfig.ConnectTimeout)
	assert.Equal(t, 11, serverConfig.ConnectionRequestTimeout)

	serverConfig = config.Servers["testServer"]
	assert.Equal(t, "localhost.prod", serverConfig.Host)
	assert.Equal(t, 9123, serverConfig.Port)
	assert.Equal(t, true, serverConfig.Https)
	assert.Equal(t, 50, serverConfig.ConnectTimeout)
	assert.Equal(t, 50, serverConfig.ConnectionRequestTimeout)

	// Test a parameterized var
	assert.Equal(t, "localhost.prod", config.Servers["testServer"].Host)
}

var dataForTestParseConfigWithParameterizedConfig_WithDev = `
env: dev

servers:
  jsonplaceholder:
    host: "env:string: prod=jsonplaceholder.typicode.com; stage=localhost.stage; default=localhost.dev"
    port: "env:int: prod=443; default=8080"
    https: true
    connect_timeout: "env:int: prod=10; default=1000"
    connection_request_timeout: "env:int: prod=11; default=1001"
  testServer:
    host: "env:string: prod=localhost.prod; dev=localhost.dev; stage=localhost.stage"
    port: 9123
    https: "env:bool: prod=true; dev=false; stage=false"

apis:
  delay_timeout_10:
    path: /delay/delay_timeout_10
    server: testServer
    timeout: "env:int: prod=10; default=1000"
    concurrency: "env:int: prod=10; default=300"
  delay_timeout_10_POST:
    path: /delay/delay_timeout_10_POST
    method: POST
    server: testServer
    timeout: "env:int: prod=100; default=1001"
    concurrency: "env:int: prod=11; default=200"
`

func TestParseConfigWithParameterizedConfig_WithDev(t *testing.T) {
	config := Config{}
	err := serialization.ReadYamlFromString(dataForTestParseConfigWithParameterizedConfig_WithDev, &config)
	assert.NoError(t, err)
	assert.True(t, len(config.Servers) > 0)
	assert.True(t, len(config.Apis) > 0)

	serverConfig := config.Servers["jsonplaceholder"]
	assert.Equal(t, "localhost.dev", serverConfig.Host)
	assert.Equal(t, 8080, serverConfig.Port)
	assert.Equal(t, true, serverConfig.Https)
	assert.Equal(t, 1000, serverConfig.ConnectTimeout)
	assert.Equal(t, 1001, serverConfig.ConnectionRequestTimeout)

	serverConfig = config.Servers["testServer"]
	assert.Equal(t, "localhost.dev", serverConfig.Host)
	assert.Equal(t, 9123, serverConfig.Port)
	assert.Equal(t, false, serverConfig.Https)
	assert.Equal(t, 50, serverConfig.ConnectTimeout)
	assert.Equal(t, 50, serverConfig.ConnectionRequestTimeout)

	api := config.Apis["delay_timeout_10"]
	assert.Equal(t, "GET", api.Method)
	assert.Equal(t, "/delay/delay_timeout_10", api.Path)
	assert.Equal(t, 1000, api.Timeout)
	assert.Equal(t, 300, api.Concurrency)

	api = config.Apis["delay_timeout_10_POST"]
	assert.Equal(t, "POST", api.Method)
	assert.Equal(t, "/delay/delay_timeout_10_POST", api.Path)
	assert.Equal(t, 1001, api.Timeout)
	assert.Equal(t, 200, api.Concurrency)
}
