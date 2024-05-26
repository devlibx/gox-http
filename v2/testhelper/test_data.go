package testhelper

import (
	_ "embed"
	"github.com/devlibx/gox-base/serialization"
)

func GetTestConfig(config interface{}) error {
	// return serialization.ReadYaml("./test_config.yaml", config)
	return serialization.ReadYamlFromString(TestConfig, config)
}

//go:embed test_config_real_server.yaml
var TestConfigWithRealServer string

var TestConfig = `
servers:
  jsonplaceholder:
    host: jsonplaceholder.typicode.com
    port: 80
    https: false
    connect_timeout: 1000
    connection_request_timeout: 1000
  testServer:
    host: localhost
    port: 9123

apis:
  getPosts:
    method: GET
    path: /posts/{id}
    server: jsonplaceholder
    timeout: 1000
    acceptable_codes: 200,201
  delay_timeout_10:
    path: /delay
    server: testServer
    timeout: 10
    concurrency: 3
  delay_timeout_20:
    path: /delay
    server: testServer
    timeout: 20
  delay_timeout_50:
    path: /delay
    server: testServer
    timeout: 50
  delay_timeout_100:
    path: /delay
    server: testServer
    timeout: 100
    concurrency: 3
  delay_timeout_1000:
    path: /delay
    server: testServer
    timeout: 1000
    concurrency: 3
    queueSize: 1
  delay_timeout_5000:
    path: /delay
    server: testServer
    timeout: 5000
    concurrency: 3
    queueSize: 1
  post_api_with_delay_2000:
    method: POST
    path: /delay
    server: testServer
    timeout: 2000
  put_api_with_delay_2000:
    method: PUT
    path: /delay
    server: testServer
    timeout: 2000
  delete_api_with_delay_2000:
    method: DELETE
    path: /delay
    server: testServer
    timeout: 2000
`
