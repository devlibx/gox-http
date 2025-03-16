package command

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpdateServerWithUrl(t *testing.T) {
	// Test cases
	testCases := []struct {
		name     string
		url      string
		expHost  string
		expPort  int
		expHttps bool
	}{
		{"HTTP with IP and Port", "http://127.0.0.1:1234", "127.0.0.1", 1234, false},
		{"HTTP with Hostname and Port", "http://localhost:1234", "localhost", 1234, false},
		{"HTTPS with Hostname and Port", "https://example.com:8443", "example.com", 8443, true},
		{"HTTP with Hostname No Port", "http://localhost", "localhost", 9999, false}, // Port should remain unchanged
		{"URL with Path", "http://api.example.com:8080/v1/users", "api.example.com", 8080, false},
		{"URL with Query Params", "https://search.example.com:9000/search?q=test", "search.example.com", 9000, true},
		{"URL with Auth", "http://user:pass@auth.example.com:8888", "auth.example.com", 8888, false},
		{"IPv6 Address", "http://[::1]:8080", "::1", 8080, false},
		{"HTTPS Default Port", "https://secure.example.com:443", "secure.example.com", 443, true},
		{"HTTP Default Port", "http://example.com:80", "example.com", 80, false},
		{"Invalid URL", "://invalid-url", "original-host", 9999, false}, // Should not change anything
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a config with a test server
			config := &Config{
				Servers: Servers{
					"testServer": &Server{
						Host:  "original-host",
						Port:  9999,
						Https: !tc.expHttps, // Set to opposite to verify change
					},
				},
			}

			// Call the method being tested
			config.UpdateServerWithUrl("testServer", tc.url)

			// Get the server after update
			server := config.Servers["testServer"]

			// Verify the results
			if tc.name == "Invalid URL" {
				assert.Equal(t, "original-host", server.Host)
				assert.Equal(t, 9999, server.Port)
				assert.Equal(t, !tc.expHttps, server.Https)
			} else {
				assert.Equal(t, tc.expHost, server.Host)
				assert.Equal(t, tc.expPort, server.Port)
				assert.Equal(t, tc.expHttps, server.Https)
			}
		})
	}
}

func TestUpdateServerWithUrl_NonExistentServer(t *testing.T) {
	config := &Config{
		Servers: Servers{},
	}

	// This should not panic or cause any errors
	config.UpdateServerWithUrl("nonExistentServer", "http://example.com:8080")

	// Verify the server wasn't created
	_, exists := config.Servers["nonExistentServer"]
	assert.False(t, exists)
}
