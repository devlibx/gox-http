package goxHttpApi

import (
	"fmt"
	"testing"
)

func TestNewRequestResponseSecurityConfigApplier(t *testing.T) {

	req := `{
    "remove_me": "1",
    "data": {
        "remove_me": "2",
        "pan": "1234",
        "good": "some",
        "sub_data": {
            "remove_me": "2",
            "pan": "1234",
            "good": "some"
        }
    }
}`

	log := &RequestResponseLog{
		URL:               "",
		RequestHeaders:    map[string]interface{}{"x-remove-me-1": 1, "x-remove-me": "2", "x-dont-remove-me": "3"},
		ResponseHeaders:   map[string]interface{}{"x-resp-remove-me-1": 1, "x-resp-remove-me": "2", "x-resp-dont-remove-me": "3"},
		RequestBodyBytes:  []byte(req),
		ResponseBodyBytes: []byte(req),
	}

	applier := NewRequestResponseSecurityConfigApplier(&RequestResponseSecurityConfig{
		EnableRequestLogging:  true,
		IgnoreRequestHeaders:  []string{"x-remove-me-1", "x-remove-me"},
		IgnoreResponseHeaders: []string{"x-resp-remove-me-1", "x-resp-remove-me"},
		IgnoreKeysInRequest:   []string{"remove_me", "pan"},
		IgnoreKeysInResponse:  []string{"remove_me", "pan"},
	})
	log = applier.Process(log)
	fmt.Println(log.String())

}
