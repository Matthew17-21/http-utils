//go:build integration
// +build integration

package httputils

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// compressionTest defines the structure for each compression test case
type compressionTest struct {
	name           string
	url            string
	acceptEncoding string
	expectedField  string
	expectedValue  bool
}

// responseData represents the common structure of httpbin.org responses
type responseData struct {
	Headers struct {
		AcceptEncoding string `json:"Accept-Encoding"`
	} `json:"headers"`
	Method string `json:"method"`
	Origin string `json:"origin"`
	// Dynamic fields for different compression types
	Brotli  bool `json:"brotli,omitempty"`
	Gzip    bool `json:"gzipped,omitempty"`
	Deflate bool `json:"deflated,omitempty"`
}

func TestDecompressResponse(t *testing.T) {
	client := &http.Client{}
	retryableClient := NewRetryableHTTPClient(client)

	// Define test cases in a table-driven approach
	testCases := []compressionTest{
		{
			name:           "brotli",
			url:            "https://httpbin.org/brotli",
			acceptEncoding: "br",
			expectedField:  "Brotli",
			expectedValue:  true,
		},
		{
			name:           "gzip",
			url:            "https://httpbin.org/gzip",
			acceptEncoding: "gzip",
			expectedField:  "Gzip",
			expectedValue:  true,
		},
		{
			name:           "deflate",
			url:            "https://httpbin.org/deflate",
			acceptEncoding: "deflate",
			expectedField:  "Deflate",
			expectedValue:  true,
		},
	}

	// Run all test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runCompressionTest(t, retryableClient, tc)
		})
	}
}

// runCompressionTest executes a single compression test case
func runCompressionTest(t *testing.T, client *RetryableHTTPClient[*http.Client, *http.Request, *http.Response], tc compressionTest) {
	// Create and configure the request
	req, err := http.NewRequest("GET", tc.url, nil)
	require.NoError(t, err)
	req.Header.Set("Accept-Encoding", tc.acceptEncoding)

	// Perform the request
	resp, err := client.DoWithRetry(context.Background(), req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Decompress the response
	decompressed, err := DecompressResponse(resp.Header, resp.Body)
	require.NoError(t, err)

	// Parse and validate the response
	validateResponse(t, decompressed, tc)
}

// validateResponse parses the JSON response and validates the expected fields
func validateResponse(t *testing.T, decompressed []byte, tc compressionTest) {
	var response responseData
	err := json.Unmarshal(decompressed, &response)
	require.NoError(t, err)

	// Validate common fields
	require.Equal(t, tc.acceptEncoding, response.Headers.AcceptEncoding)
	require.Equal(t, "GET", response.Method)
	require.NotEmpty(t, response.Origin)

	// Validate compression-specific field
	switch tc.expectedField {
	case "Brotli":
		require.True(t, response.Brotli)
	case "Gzip":
		require.True(t, response.Gzip)
	case "Deflate":
		require.True(t, response.Deflate)
	default:
		t.Fatalf("Unknown expected field: %s", tc.expectedField)
	}
}
