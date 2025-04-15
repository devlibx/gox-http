# Gox Http

[![Go Reference](https://pkg.go.dev/badge/github.com/devlibx/gox-http.svg)](https://pkg.go.dev/github.com/devlibx/gox-http)

A robust HTTP client library for Go that provides advanced features like circuit breaking, concurrency control, retries, and environment-specific configurations.

## Key Features

- 🔧 **Configuration-Driven**: Define all endpoints and API configs in YAML
- 🛡️ **Circuit Breaking**: Built-in Hystrix support for fault tolerance
- 🚦 **Concurrency Control**: Set parallel request limits and queue size per API
- 🎯 **Status Code Handling**: Define acceptable status codes with custom error handling
- 🔄 **Retry Support**: Configurable retries with custom wait times
- 📨 **Header Management**: Server-level and API-specific headers with context propagation
- ⏱️ **Timeout Management**: Configure timeouts at API level
- 🌍 **Environment Support**: Environment-specific configurations
- 🔐 **HMAC Authentication**: Built-in HMAC SHA256 validation
- 🔍 **Request/Response Logging**: Detailed logging capabilities
- 📡 **Proxy Support**: Configure proxy settings per server
- 🔒 **TLS Configuration**: Skip certificate verification options

## Installation

```bash
go get github.com/devlibx/gox-http/v4
```

## Usage Examples

### Basic Request

```go
request := command.NewGoxRequestBuilder("getPosts").
    WithContentTypeJson().
    WithPathParam("id", "123").
    WithQueryParam("filter", "active").
    WithResponseBuilder(command.NewJsonToObjectResponseBuilder(&gox.StringObjectMap{})).
    Build()

response, err := goxHttpCtx.Execute(context.Background(), "getPosts", request)
```

### Async Request

```go
request := command.NewGoxRequestBuilder("asyncApi").Build()
responseChannel := goxHttpCtx.ExecuteAsync(context.Background(), "asyncApi", request)
response := <-responseChannel
```

### Error Handling

```go
response, err := goxHttpCtx.Execute(context.Background(), "getPosts", request)
if err != nil {
    if goxError, ok := err.(*command.GoxHttpError); ok {
        switch {
        case goxError.Is5xx():
            // Handle 5xx errors
        case goxError.Is4xx():
            // Handle 4xx errors
        case goxError.IsHystrixCircuitOpenError():
            // Handle circuit breaker open
        case goxError.IsHystrixTimeoutError():
            // Handle timeouts
        case goxError.IsHystrixRejectedError():
            // Handle rejection due to high concurrency
        }
    }
}
```

## Configuration Guide

### Complete Example

```yaml
servers:
  jsonplaceholder:
    host: jsonplaceholder.typicode.com
    port: 443
    https: true
    connect_timeout: 1000
    connection_request_timeout: 1000
    headers:  # Server-level headers
      Authorization: "Bearer default-token"
      X-API-Version: "1.0"
      __UNIQUE_UUID__: true  # Auto-generates unique UUID per request
    properties:
      mdc: "trace-id,request-id"  # Propagate context values as headers
  testServer:
    host: localhost
    port: 9123
    headers:
      X-Client-ID: "test-client"
      X-Environment: "local"

apis:
  getPosts:
    method: GET
    path: /posts/{id}
    server: jsonplaceholder
    timeout: 1000
    acceptable_codes: 200,201
    concurrency: 10  # Max parallel requests
    queue_size: 100  # Request queue size
    retry_count: 3
    retry_initial_wait_time_ms: 100
    headers:  # API-specific headers (overrides server headers)
      Content-Type: "application/json"
      X-Custom-Header: "custom-value"
  delay_timeout_10:
    path: /delay
    server: testServer
    timeout: 10
    concurrency: 3
    headers:
      Authorization: "Bearer specific-token"  # Override server's auth token
```

```go
package main

import (
    "context"
    "fmt"
    "github.com/devlibx/gox-base"
    "github.com/devlibx/gox-base/serialization"
    goxHttpApi "github.com/devlibx/gox-http/v4/api"
    "github.com/devlibx/gox-http/v4/command"
    "log"
)

func main() {
    cf := gox.NewCrossFunction()

    // Read config
    config := command.Config{}
    err := serialization.ReadYamlFromString(httpConfig, &config)
    if err != nil {
        log.Println("got error in reading config", err)
        return
    }

    // Setup goHttp context
    goxHttpCtx, err := goxHttpApi.NewGoxHttpContext(cf, &config)
    if err != nil {
        log.Println("got error in creating gox http context config", err)
        return
    }

    // Make a http call and get the result
    request := command.NewGoxRequestBuilder("getPosts").
        WithContentTypeJson().
        WithPathParam("id", 1).
        WithResponseBuilder(command.NewJsonToObjectResponseBuilder(&gox.StringObjectMap{})).
        Build()
    response, err := goxHttpCtx.Execute(context.Background(), "getPosts", request)
    if err != nil {
        fmt.Println("got error", err)
        return
    }
    fmt.Println(serialization.Stringify(response.Response))
}
```

### Server Configuration

#### Server Configuration Properties

| Property | Description | Default | Required |
|----------|-------------|---------|----------|
| host | Server hostname | - | Yes |
| port | Server port | - | Yes |
| https | Use HTTPS | false | No |
| proxy_url | Proxy server URL | - | No |
| skip_cert_verify | Skip TLS verification | false | No |
| connect_timeout | Connection timeout (ms) | 1000 | No |
| connection_request_timeout | Request timeout (ms) | 1000 | No |
| enable_http_connection_tracing | Enable connection tracing | false | No |
| headers | Server-level headers | - | No |
| properties | Custom properties map | - | No |
| interceptor_config | Interceptor configuration | - | No |

### Header Management

Headers can be configured at both server and API levels, with API-level headers taking precedence over server-level headers.

#### Server-Level Headers
```yaml
servers:
  my_server:
    headers:
      Authorization: "Bearer ${TOKEN}"  # Static header
      __UNIQUE_UUID__: true  # Auto-generated UUID per request
      X-API-Version: "2.0"  # Version header
    properties:
      mdc: "trace-id,request-id,correlation-id"  # Context propagation
```

#### API-Level Headers
```yaml
apis:
  get_user:
    method: GET
    path: /users/{id}
    server: my_server
    timeout: 1000
    concurrency: 10
    acceptable_codes: "200,201,404"
    retry_count: 3
    retry_initial_wait_time_ms: 100
    headers:  # Override or add to server headers
      Content-Type: "application/json"
      Authorization: "Bearer ${API_TOKEN}"  # Override server's auth
      X-Operation-ID: "get-user"  # API-specific header
```

#### Header Features
- **Server-Level Default Headers**: Set default headers for all APIs using a server
- **API-Level Overrides**: Override or add to server headers for specific APIs
- **Special Headers**:
  - `__UNIQUE_UUID__`: Set to `true` to auto-generate a unique UUID for each request
  - `Content-Type`: Defaults to `application/json` if not specified
- **Context Propagation**: Use `mdc` property to automatically propagate context values as headers
- **Dynamic Headers**: Headers can be added/modified programmatically:
```go
request := command.NewGoxRequestBuilder("getPosts").
    WithHeader("X-Custom-Header", "custom-value").
    WithHeader("Authorization", fmt.Sprintf("Bearer %s", token)).
    Build()
```

### API Configuration
```yaml
apis:
  get_user:
    method: GET
    path: /users/{id}
    server: my_server
    timeout: 1000
    concurrency: 10
    acceptable_codes: "200,201,404"
    retry_count: 3
    retry_initial_wait_time_ms: 100
    headers:
      Content-Type: "application/json"
```

#### API Configuration Properties

| Property | Description | Default | Required |
|----------|-------------|---------|----------|
| method | HTTP method (GET/POST/etc) | GET | No |
| path | API endpoint path | - | Yes |
| server | Server reference | - | Yes |
| timeout | Request timeout (ms) | 1000 | No |
| concurrency | Max parallel requests | 10 | No |
| queue_size | Request queue size | 100 | No |
| async | Enable async execution | false | No |
| acceptable_codes | Acceptable HTTP status codes | "200" | No |
| retry_count | Number of retries | 0 | No |
| retry_initial_wait_time_ms | Initial retry wait time (ms) | 0 | No |
| enable_request_response_logging | Enable request/response logging | false | No |
| enable_http_connection_tracing | Enable connection tracing | false | No |
| disable_hystrix | Disable circuit breaker | false | No |
| headers | API-specific headers | - | No |
| interceptor_config | API-level interceptor config | - | No |

### Environment-Specific Configuration

Support for environment-specific values using the `env` prefix:

```yaml
env: dev  # Environment selector (default: prod)

servers:
  api:
    host: "env:string: prod=api.prod.com; dev=api.dev.com; default=api.local"
    port: "env:int: prod=443; default=8080"
    https: "env:bool: prod=true; default=false"
    connect_timeout: "env:int: prod=1000; dev=2000; default=5000"
```

### HMAC Authentication

Configure HMAC SHA256 validation at server or API level:

```yaml
interceptor_config:
  hmac_config:
    key: "your-secret-key"
    hash_header_key: "X-Hash-Code"
    timestamp_header_key: "X-Timestamp"
    headers_to_include_in_signature: 
      - "x-custom-header"
    convert_header_keys_to_lower_case: true
```

### Dynamic API Updates

```go
// Update existing API
config.Apis["existing_api"].Path = "/new_path"
err = goxHttpCtx.ReloadApi("existing_api")

// Add new API
config.Apis["new_api"] = &command.Api{
    Name:        "new_api",
    Method:      "GET",
    Path:        "/new_endpoint",
    Server:      "testServer",
    Timeout:     100,
    Concurrency: 10,
}
err = goxHttpCtx.ReloadApi("new_api")
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
