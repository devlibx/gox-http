## Gox Http

Gox Http provides utility to call a http endpoint. It provides following:

1. Define all endpoint and api config in configuration file
2. Circuit breaker using Hystrix
3. Set concurrency for each api - this ensures that if we go beyond "concurrency" no of parallel requests then hystrix
   will reject the requests
4. Set timeout for each api - the call will timeout if this request takes time > timeout defined
5. acceptable_codes - list of "," separated status codes which are acceptable. These status codes will not be counted as
   errors and will not open hystrix circuit

#### How to use

Given below is a example on how to use this liberary

```go
package main

import (
	"context"
	"fmt"
	"github.com/devlibx/gox-base"
	"github.com/devlibx/gox-base/serialization"
	goxHttpApi "github.com/devlibx/gox-http/api"
	"github.com/devlibx/gox-http/command"
	"log"
)

// Here you can define your own configuration
// We have used "jsonplaceholder" as a test server. A api "getPosts" is defined which uses "server=jsonplaceholder"
var httpConfig = `
servers:
  jsonplaceholder:
    host: jsonplaceholder.typicode.com
    port: 443
    https: true
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
`

func main() {

	cf := gox.NewCrossFunction()

	// Read config and
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
	// 	ResponseBuilder - this is used to convert json response to your custom object
	//
	//  The following interface can be implemented to convert from bytes to the desired output.
	//  response.Response will hold the object which is returned from  ResponseBuilder
	//
	//	type ResponseBuilder interface {
	//		Response(data []byte) (interface{}, error)
	//	}
	request := command.NewGoxRequestBuilder("getPosts").
		WithContentTypeJson().
		WithPathParam("id", 1).
		WithResponseBuilder(command.NewJsonToObjectResponseBuilder(&gox.StringObjectMap{})).
		Build()
	response, err := goxHttpCtx.Execute(context.Background(), "getPosts", request)
	if err != nil {

		// Error details can be extracted from *command.GoxHttpError
		if goxError, ok := err.(*command.GoxHttpError); ok {
			if goxError.Is5xx() {
				fmt.Println("got 5xx error")
			} else if goxError.Is4xx() {
				fmt.Println("got 5xx error")
			} else if goxError.IsBadRequest() {
				fmt.Println("got bad request error")
			} else if goxError.IsHystrixCircuitOpenError() {
				fmt.Println("hystrix circuit is open due to many errors")
			} else if goxError.IsHystrixTimeoutError() {
				fmt.Println("hystrix timeout because http call took longer then configured")
			} else if goxError.IsHystrixRejectedError() {
				fmt.Println("hystrix rejected the request because too many concurrent request are made")
			} else if goxError.IsHystrixError() {
				fmt.Println("hystrix error - timeout/circuit open/rejected")
			}

		} else {
			fmt.Println("got unknown error")
		}

	} else {
		fmt.Println(serialization.Stringify(response.Response))
		// {some json response ...}
	}
}

```

#### Retry Handling

You can specify following properties in a API to enable a retry.

1. retry_count - how many times you want to retry
2. retry_initial_wait_time_ms - a delay before making a retry
3. NOTE - the total Hystrix timeout will be set to (timeout + (retry_count * timeout) + retry_initial_wait_time_ms)
   <br> Timeout is the time taken by a single call. So the total time is adjusted to cover retries
4. If response from a server is an acceptable code then retry will not be done e.g. in this case status=404 will not
   trigger a retry.

```yaml
apis:
  getPosts:
    method: GET
    path: /posts/{id}
    server: jsonplaceholder
    timeout: 1000
    acceptable_codes: 200,201,404
    retry_count: 3
    retry_initial_wait_time_ms: 10
```
----
## Environment Specific Configs Support
You can setup all properties with env specific values
1. env = name of the env (default=prod). This is used to find the values for all properties
2. add "env: " in front of all values to make it configurable
3. setup env specific configs
4. <b> Note - You must use serialization.ReadParameterizedYaml() method if you uses parameterized yaml</b>
```yaml
host: "env:string: prod=localhost.prod; dev=localhost.dev; stage=localhost.stage"
```
Here host value will be based on the "env" you have provided in a config. For example host will be 
"localhost.prod" if env=prod, or host="localhost.stage" if env=stage"
4. Default: You can sprcify "default" - if no value match this will be used
<br>
   e.g. ```port: "env:int: prod=443; default=8080"``` dev/stage/any other will pick port=8080. Only prod will use 443 


```yaml
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
    https: "env:int: prod=true; dev=false; stage=false"

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
```
---

### How to add or update a new API dynamically
NOTE - we only support adding new API (new server has not be done manually)

```go
// Load config from test_config_real_server.yaml example file
config := command.Config{}
err := serialization.ReadYamlFromString(testhelper.TestConfigWithRealServer, &config)
	
// Suppose a API "delay_timeout_10" was using a path "/delay", and you want to chnage it to point to "/delay_new"
// Update the config and reload API
config.Apis["delay_timeout_10"].Path = "/delay_new"
err = goxHttpCtx.ReloadApi("delay_timeout_10")


// Add a new API inimically
config.Apis["new_api"] = &command.Api{
    Name:        "new_api",
    Method:      "GET",
    Path:        "/bad_new",
    Server:      "testServer",
    Timeout:     100,
    Concurrency: 10,
    QueueSize:   10,
}
err = goxHttpCtx.ReloadApi("new_api")

request = command.NewGoxRequestBuilder("new_api").
                WithContentTypeJson().
               WithPathParam("id", 1).
               WithResponseBuilder(command.NewJsonToObjectResponseBuilder(&gox.StringObjectMap{})).
               Build()
response, err = goxHttpCtx.Execute(ctx, "new_api", request)
assert.NoError(t, err)
assert.Equal(t, "ok", response.AsStringObjectMapOrEmpty().StringOrEmpty("status"))
assert.Equal(t, "/bad_new", response.AsStringObjectMapOrEmpty().StringOrEmpty("url"))

```

### Enable HmacSha256 validation
1. interceptor_config.hmac_config => set this up to use HMAC SHA256
2. key => secret key to use for HMAC
3. hash_header_key => Hash will be calculated and will be passed with this header key
4. timestamp_header_key => timestamp will be passed with this header key
5. headers_to_include_in_signature => list of headers which will be included in signature calculation
6. convert_header_keys_to_lower_case => if true then all header keys will be converted to lower case before calculating signature
```yaml
servers:
  jsonplaceholder:
    host: jsonplaceholder.typicode.com
    port: 443
    https: true
    interceptor_config:
       hmac_config:
          key: <some secret>
          hash_header_key: X-Hash-code-sha
          timestamp_header_key: X-Time
          headers_to_include_in_signature: [x-header-1, x-header-2]
          convert_header_keys_to_lower_case: true
```

#### Error Handling
You can register `RequestValidatorErrorHandlingMiddleware` in your router to get common error handler for bad request.
```go
type User struct {
    Name       string   `json:"name" binding:"required,max=10"`
    Email      string   `json:"email" binding:"required,email"`
    Address    Address  `json:"address" binding:"required"`
    AddressPtr *Address `json:"address_ptr" binding:"required"`
}

type Address struct {
    FlatNo int `json:"flat_no" binding:"required"`
}

// Register middleware to handle bad request
r := gin.Default()
r.Use(RequestValidatorErrorHandlingMiddleware(DoNotSkipRequestValidationFunc, nil))

// Use this middleware to handle error 
var user User
if err := c.ShouldBindJSON(&user); err != nil {
    _ = c.AbortWithError(http.StatusBadRequest, err)
}
```

```json
{
    "status": 400,
    "error_mapping_info": {
        "User.Address.FlatNo": "Error:Field validation for 'FlatNo' failed on the 'required' tag",
        "User.AddressPtr": "Error:Field validation for 'AddressPtr' failed on the 'required' tag",
        "User.Email": "Error:Field validation for 'Email' failed on the 'required' tag"
    }
}
```