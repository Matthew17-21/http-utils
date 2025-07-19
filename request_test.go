package httputils

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"testing"
	"time"

	tlsHttp "github.com/bogdanfinn/fhttp"
	"github.com/stretchr/testify/require"
)

// Mock HTTP client for standard http.Request
type mockStandardHTTPClient struct {
	responses    []*http.Response
	errors       []error
	callCount    int
	requestsMade []*http.Request
}

func (m *mockStandardHTTPClient) Do(req *http.Request) (*http.Response, error) {
	m.requestsMade = append(m.requestsMade, req)
	defer func() { m.callCount++ }()

	if m.callCount < len(m.errors) && m.errors[m.callCount] != nil {
		return nil, m.errors[m.callCount]
	}
	if m.callCount < len(m.responses) {
		return m.responses[m.callCount], nil
	}
	return nil, errors.New("unexpected call")
}

// Mock HTTP client for TLS http.Request
type mockTLSHTTPClient struct {
	responses    []*tlsHttp.Response
	errors       []error
	callCount    int
	requestsMade []*tlsHttp.Request
}

func (m *mockTLSHTTPClient) Do(req *tlsHttp.Request) (*tlsHttp.Response, error) {
	m.requestsMade = append(m.requestsMade, req)
	defer func() { m.callCount++ }()

	if m.callCount < len(m.errors) && m.errors[m.callCount] != nil {
		return nil, m.errors[m.callCount]
	}
	if m.callCount < len(m.responses) {
		return m.responses[m.callCount], nil
	}
	return nil, errors.New("unexpected call")
}

// Mock logger for testing
type mockLogger struct {
	debugLogs []string
	infoLogs  []string
	warnLogs  []string
	errorLogs []string
}

func (m *mockLogger) Debug(format string, args ...any) {
	m.debugLogs = append(m.debugLogs, fmt.Sprintf(format, args...))
}

func (m *mockLogger) Info(format string, args ...any) {
	m.infoLogs = append(m.infoLogs, fmt.Sprintf(format, args...))
}

func (m *mockLogger) Warn(format string, args ...any) {
	m.warnLogs = append(m.warnLogs, fmt.Sprintf(format, args...))
}

func (m *mockLogger) Error(format string, args ...any) {
	m.errorLogs = append(m.errorLogs, fmt.Sprintf(format, args...))
}

func (m *mockLogger) reset() {
	m.debugLogs = nil
	m.infoLogs = nil
	m.warnLogs = nil
	m.errorLogs = nil
}

func TestNewRetryableHTTPClient(t *testing.T) {
	t.Run("creates client with default options", func(t *testing.T) {
		mockClient := &mockStandardHTTPClient{}
		client := NewRetryableHTTPClient(mockClient)

		require.NotNil(t, client)
		require.Equal(t, mockClient, client.client)
		require.Equal(t, 1, client.numRetries)
		require.NotNil(t, client.logger)
	})

	t.Run("creates client with custom options", func(t *testing.T) {
		mockClient := &mockStandardHTTPClient{}
		mockLogger := &mockLogger{}
		client := NewRetryableHTTPClient(
			mockClient,
			WithRetries[*mockStandardHTTPClient](5),
			WithLogger[*mockStandardHTTPClient](mockLogger),
		)

		require.NotNil(t, client)
		require.Equal(t, mockClient, client.client)
		require.Equal(t, 5, client.numRetries)
		require.Equal(t, mockLogger, client.logger)
	})

	t.Run("creates TLS client", func(t *testing.T) {
		mockClient := &mockTLSHTTPClient{}
		client := NewRetryableHTTPClient(mockClient)

		require.NotNil(t, client)
		require.Equal(t, mockClient, client.client)
		require.Equal(t, 1, client.numRetries)
		require.NotNil(t, client.logger)
	})
}

func TestWithRetries(t *testing.T) {
	mockClient := &mockStandardHTTPClient{}

	tests := []struct {
		name     string
		retries  int
		expected int
	}{
		{"zero retries", 0, 0},
		{"one retry", 1, 1},
		{"multiple retries", 10, 10},
		{"negative retries", -1, -1}, // Should still set it even if invalid
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := NewRetryableHTTPClient(
				mockClient,
				WithRetries[*mockStandardHTTPClient](test.retries),
			)
			require.Equal(t, test.expected, client.numRetries)
		})
	}
}

func TestWithLogger(t *testing.T) {
	mockClient := &mockStandardHTTPClient{}
	mockLogger := &mockLogger{}

	client := NewRetryableHTTPClient(
		mockClient,
		WithLogger[*mockStandardHTTPClient](mockLogger),
	)

	require.Equal(t, mockLogger, client.logger)
}

func TestMakeRequestSuccessful(t *testing.T) {
	t.Run("successful request on first attempt", func(t *testing.T) {
		mockClient := &mockStandardHTTPClient{
			responses: []*http.Response{
				{StatusCode: 200},
			},
		}
		mockLogger := &mockLogger{}

		client := NewRetryableHTTPClient(
			mockClient,
			WithLogger[*mockStandardHTTPClient](mockLogger),
		)

		req, _ := http.NewRequest("GET", "http://example.com", nil)
		ctx := context.Background()

		resp, err := client.MakeRequest(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, 200, resp.StatusCode)
		require.Equal(t, 1, mockClient.callCount)

		// Check that debug logs were created
		require.Contains(t, strings.Join(mockLogger.debugLogs, " "), "Starting DoWithRetry")
		require.Contains(t, strings.Join(mockLogger.debugLogs, " "), "Request successful")
	})

	t.Run("successful TLS request", func(t *testing.T) {
		mockClient := &mockTLSHTTPClient{
			responses: []*tlsHttp.Response{
				{StatusCode: 200},
			},
		}

		client := NewRetryableHTTPClient(mockClient)

		req, _ := tlsHttp.NewRequest("GET", "http://example.com", nil)
		ctx := context.Background()

		resp, err := client.MakeRequest(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, 200, resp.StatusCode)
		require.Equal(t, 1, mockClient.callCount)
	})
}

func TestMakeRequestRetries(t *testing.T) {
	t.Run("retries on retryable error and succeeds", func(t *testing.T) {
		mockClient := &mockStandardHTTPClient{
			errors: []error{
				&url.Error{
					Op:  "Get",
					URL: "http://example.com",
					Err: &net.OpError{Op: "dial", Err: syscall.ETIMEDOUT},
				},
				nil, // Second attempt succeeds
			},
			responses: []*http.Response{
				nil,               // First attempt fails
				{StatusCode: 200}, // Second attempt succeeds
			},
		}
		mockLogger := &mockLogger{}

		client := NewRetryableHTTPClient(
			mockClient,
			WithRetries[*mockStandardHTTPClient](2),
			WithLogger[*mockStandardHTTPClient](mockLogger),
		)

		req, _ := http.NewRequest("GET", "http://example.com", nil)
		ctx := context.Background()

		resp, err := client.MakeRequest(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, 200, resp.StatusCode)
		require.Equal(t, 2, mockClient.callCount)

		// Check that error was logged
		require.True(t, len(mockLogger.errorLogs) > 0)
		require.Contains(t, mockLogger.errorLogs[0], "Retrying")
	})

	t.Run("exhausts all retries and fails", func(t *testing.T) {
		retryableError := &url.Error{
			Op:  "Get",
			URL: "http://example.com",
			Err: &net.OpError{Op: "dial", Err: syscall.ETIMEDOUT},
		}

		mockClient := &mockStandardHTTPClient{
			errors: []error{
				retryableError,
				retryableError,
				retryableError,
			},
		}
		mockLogger := &mockLogger{}

		client := NewRetryableHTTPClient(
			mockClient,
			WithRetries[*mockStandardHTTPClient](3),
			WithLogger[*mockStandardHTTPClient](mockLogger),
		)

		req, _ := http.NewRequest("GET", "http://example.com", nil)
		ctx := context.Background()

		resp, err := client.MakeRequest(ctx, req)

		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "failed to make request after 3 attempts")
		require.Equal(t, 3, mockClient.callCount)

		// Check that errors were logged
		require.Equal(t, 3, len(mockLogger.errorLogs))
	})
}

func TestMakeRequestNonRetryableErrors(t *testing.T) {
	tests := []struct {
		name  string
		error error
	}{
		{
			name: "BadURL error",
			error: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: errors.New("invalid URL scheme: []"),
			},
		},
		{
			name: "ProxyAuthRequired error",
			error: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: errors.New("Proxy responded with non 200 code: 407 Proxy Authentication Required"),
			},
		},
		{
			name: "ProxyRelayOffline error",
			error: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: errors.New("Proxy responded with non 200 code: 502 Proxy Error (The selected relay is offline or busy processing other threads)"),
			},
		},
		{
			name: "DNSNameNotFound error",
			error: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: &net.DNSError{IsNotFound: true},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockClient := &mockStandardHTTPClient{
				errors: []error{test.error},
			}
			mockLogger := &mockLogger{}

			client := NewRetryableHTTPClient(
				mockClient,
				WithRetries[*mockStandardHTTPClient](3),
				WithLogger[*mockStandardHTTPClient](mockLogger),
			)

			req, _ := http.NewRequest("GET", "http://example.com", nil)
			ctx := context.Background()

			resp, err := client.MakeRequest(ctx, req)

			require.Error(t, err)
			require.Nil(t, resp)
			require.Equal(t, 1, mockClient.callCount) // Should not retry

			// Check that the debug log indicates immediate return
			debugLog := strings.Join(mockLogger.debugLogs, " ")
			require.Contains(t, debugLog, "returning immediately")
		})
	}
}

func TestMakeRequestContextCancellation(t *testing.T) {
	t.Run("context cancelled during retry", func(t *testing.T) {
		retryableError := &url.Error{
			Op:  "Get",
			URL: "http://example.com",
			Err: &net.OpError{Op: "dial", Err: syscall.ETIMEDOUT},
		}

		mockClient := &mockStandardHTTPClient{
			errors: []error{retryableError, retryableError},
		}
		mockLogger := &mockLogger{}

		client := NewRetryableHTTPClient(
			mockClient,
			WithRetries[*mockStandardHTTPClient](3),
			WithLogger[*mockStandardHTTPClient](mockLogger),
		)

		req, _ := http.NewRequest("GET", "http://example.com", nil)
		ctx, cancel := context.WithCancel(context.Background())

		// Cancel context immediately after starting the request
		cancel()

		resp, err := client.MakeRequest(ctx, req)

		require.Error(t, err)
		require.Nil(t, resp)

		// Should not have made any attempts due to immediate context cancellation
		require.Equal(t, 0, mockClient.callCount)

		// Check that context cancellation was logged
		debugLog := strings.Join(mockLogger.debugLogs, " ")
		require.Contains(t, debugLog, "Context is done before attempt")
	})

	t.Run("context timeout", func(t *testing.T) {
		retryableError := &url.Error{
			Op:  "Get",
			URL: "http://example.com",
			Err: &net.OpError{Op: "dial", Err: syscall.ETIMEDOUT},
		}

		mockClient := &mockStandardHTTPClient{
			errors: []error{retryableError, retryableError},
		}

		client := NewRetryableHTTPClient(
			mockClient,
			WithRetries[*mockStandardHTTPClient](3),
		)

		req, _ := http.NewRequest("GET", "http://example.com", nil)
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		// Add a small delay to ensure timeout
		time.Sleep(5 * time.Millisecond)

		resp, err := client.MakeRequest(ctx, req)

		require.Error(t, err)
		require.Nil(t, resp)
	})
}

func TestHandleRequestError(t *testing.T) {
	mockClient := &mockStandardHTTPClient{}
	mockLogger := &mockLogger{}
	client := NewRetryableHTTPClient(
		mockClient,
		WithLogger[*mockStandardHTTPClient](mockLogger),
	)

	tests := []struct {
		name           string
		error          error
		expectedReturn bool
		description    string
	}{
		{
			name:           "BadURL error",
			error:          &url.Error{Op: "Get", URL: "http://example.com", Err: errors.New("invalid URL scheme: []")},
			expectedReturn: true,
			description:    "should return immediately for BadURL errors",
		},
		{
			name:           "ProxyAuthRequired error",
			error:          &url.Error{Op: "Get", URL: "http://example.com", Err: errors.New("Proxy responded with non 200 code: 407 Proxy Authentication Required")},
			expectedReturn: true,
			description:    "should return immediately for proxy auth errors",
		},
		{
			name: "DNSNameNotFound error",
			error: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: &net.DNSError{IsNotFound: true},
			},
			expectedReturn: true,
			description:    "should return immediately for DNS not found errors",
		},
		{
			name:           "retryable timeout error",
			error:          &url.Error{Op: "Get", URL: "http://example.com", Err: &net.OpError{Op: "dial", Err: syscall.ETIMEDOUT}},
			expectedReturn: false,
			description:    "should continue retrying for timeout errors",
		},
		{
			name:           "retryable connection refused error",
			error:          &url.Error{Op: "Get", URL: "http://example.com", Err: &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED}},
			expectedReturn: false,
			description:    "should continue retrying for connection refused errors",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockLogger.reset()
			result := client.handleRequestError(context.Background(), test.error, 1)

			require.Equal(t, test.expectedReturn, result, test.description)

			// Verify appropriate logs were generated
			if result {
				debugLog := strings.Join(mockLogger.debugLogs, " ")
				require.True(t,
					strings.Contains(debugLog, "returning immediately") ||
						strings.Contains(debugLog, "Context is done before attempt"),
					"Expected immediate return to be logged")
			} else {
				require.True(t, len(mockLogger.errorLogs) > 0, "Expected error to be logged for retryable errors")
				require.Contains(t, mockLogger.errorLogs[0], "Retrying")
			}
		})
	}
}

func TestMakeRequestEdgeCases(t *testing.T) {
	t.Run("zero retries", func(t *testing.T) {
		mockClient := &mockStandardHTTPClient{
			errors: []error{errors.New("some error")},
		}

		client := NewRetryableHTTPClient(
			mockClient,
			WithRetries[*mockStandardHTTPClient](0),
		)

		req, _ := http.NewRequest("GET", "http://example.com", nil)
		ctx := context.Background()

		resp, err := client.MakeRequest(ctx, req)

		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "failed to make request after 0 attempts")
		require.Equal(t, 0, mockClient.callCount)
	})

	t.Run("negative retries", func(t *testing.T) {
		mockClient := &mockStandardHTTPClient{
			errors: []error{errors.New("some error")},
		}

		client := NewRetryableHTTPClient(
			mockClient,
			WithRetries[*mockStandardHTTPClient](-1),
		)

		req, _ := http.NewRequest("GET", "http://example.com", nil)
		ctx := context.Background()

		resp, err := client.MakeRequest(ctx, req)

		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "failed to make request after -1 attempts")
		require.Equal(t, 0, mockClient.callCount)
	})

	t.Run("nil request", func(t *testing.T) {
		mockClient := &mockStandardHTTPClient{
			responses: []*http.Response{{StatusCode: 200}},
		}

		client := NewRetryableHTTPClient(mockClient)
		ctx := context.Background()

		// This should still work as the mock client doesn't validate the request
		resp, err := client.MakeRequest(ctx, nil)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, 1, len(mockClient.requestsMade))
		require.Nil(t, mockClient.requestsMade[0])
	})
}

func TestDefaultClientOptions(t *testing.T) {
	mockClient := &mockStandardHTTPClient{}
	opts := defaultClientOptions[*mockStandardHTTPClient]()

	require.Equal(t, 2, len(opts), "Expected 2 default options")

	// Apply options to a client to verify they work
	client := &RetryableHTTPClient[*mockStandardHTTPClient, *http.Request, *http.Response]{
		client: mockClient,
	}

	for _, opt := range opts {
		opt(client)
	}

	require.Equal(t, 1, client.numRetries)
	require.NotNil(t, client.logger)
}

func TestClientOptionsCombination(t *testing.T) {
	t.Run("multiple options override correctly", func(t *testing.T) {
		mockClient := &mockStandardHTTPClient{}
		logger1 := &mockLogger{}
		logger2 := &mockLogger{}

		client := NewRetryableHTTPClient(
			mockClient,
			WithRetries[*mockStandardHTTPClient](3),
			WithLogger[*mockStandardHTTPClient](logger1),
			WithRetries[*mockStandardHTTPClient](5),      // Should override previous
			WithLogger[*mockStandardHTTPClient](logger2), // Should override previous
		)

		require.Equal(t, 5, client.numRetries)
		require.Equal(t, logger2, client.logger)
	})
}

// Integration test that combines multiple scenarios
func TestMakeRequestIntegration(t *testing.T) {
	t.Run("complex retry scenario", func(t *testing.T) {
		// Setup: First request times out, second has connection refused, third succeeds
		mockClient := &mockStandardHTTPClient{
			errors: []error{
				&url.Error{Op: "Get", URL: "http://example.com", Err: &net.OpError{Op: "dial", Err: syscall.ETIMEDOUT}},
				&url.Error{Op: "Get", URL: "http://example.com", Err: &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED}},
				nil, // Third attempt succeeds
			},
			responses: []*http.Response{
				nil, nil, // First two fail
				{StatusCode: 200}, // Third succeeds
			},
		}
		mockLogger := &mockLogger{}

		client := NewRetryableHTTPClient(
			mockClient,
			WithRetries[*mockStandardHTTPClient](3),
			WithLogger[*mockStandardHTTPClient](mockLogger),
		)

		req, _ := http.NewRequest("GET", "http://example.com", nil)
		ctx := context.Background()

		resp, err := client.MakeRequest(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, 200, resp.StatusCode)
		require.Equal(t, 3, mockClient.callCount)

		// Verify all requests were the same
		for _, capturedReq := range mockClient.requestsMade {
			require.Equal(t, req, capturedReq)
		}

		// Verify error logs for the two failed attempts
		require.Equal(t, 2, len(mockLogger.errorLogs))
		require.Contains(t, mockLogger.errorLogs[0], "Retrying")
		require.Contains(t, mockLogger.errorLogs[1], "Retrying")
	})
}

// Benchmark tests
func BenchmarkMakeRequestSuccess(b *testing.B) {
	mockClient := &mockStandardHTTPClient{
		responses: make([]*http.Response, b.N),
	}
	for i := 0; i < b.N; i++ {
		mockClient.responses[i] = &http.Response{StatusCode: 200}
	}

	client := NewRetryableHTTPClient(mockClient)
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mockClient.callCount = 0 // Reset for each iteration
		_, err := client.MakeRequest(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMakeRequestWithRetries(b *testing.B) {
	retryableError := &url.Error{
		Op:  "Get",
		URL: "http://example.com",
		Err: &net.OpError{Op: "dial", Err: syscall.ETIMEDOUT},
	}

	mockClient := &mockStandardHTTPClient{}
	for i := 0; i < b.N; i++ {
		mockClient.errors = append(mockClient.errors, retryableError, retryableError)
		mockClient.responses = append(mockClient.responses, nil, &http.Response{StatusCode: 200})
	}

	client := NewRetryableHTTPClient(
		mockClient,
		WithRetries[*mockStandardHTTPClient](2),
	)
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mockClient.callCount = i * 2 // Adjust for multiple calls per iteration
		_, err := client.MakeRequest(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Tests for the standalone DoWithRetry function
func TestDoWithRetry(t *testing.T) {
	t.Run("successful request on first attempt", func(t *testing.T) {
		mockClient := &mockStandardHTTPClient{
			responses: []*http.Response{
				{StatusCode: 200},
			},
		}
		mockLogger := &mockLogger{}

		req, _ := http.NewRequest("GET", "http://example.com", nil)
		ctx := context.Background()

		resp, err := DoWithRetry(ctx, mockClient, req, 1, mockLogger)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, 200, resp.StatusCode)
		require.Equal(t, 1, mockClient.callCount)

		// Check that debug logs were created
		require.Contains(t, strings.Join(mockLogger.debugLogs, " "), "Starting DoWithRetry")
		require.Contains(t, strings.Join(mockLogger.debugLogs, " "), "Request successful")
	})

	t.Run("successful TLS request", func(t *testing.T) {
		mockClient := &mockTLSHTTPClient{
			responses: []*tlsHttp.Response{
				{StatusCode: 200},
			},
		}

		req, _ := tlsHttp.NewRequest("GET", "http://example.com", nil)
		ctx := context.Background()

		resp, err := DoWithRetry(ctx, mockClient, req, 1, nil) // Using nil logger

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, 200, resp.StatusCode)
		require.Equal(t, 1, mockClient.callCount)
	})

	t.Run("retries on retryable error and succeeds", func(t *testing.T) {
		mockClient := &mockStandardHTTPClient{
			errors: []error{
				&url.Error{
					Op:  "Get",
					URL: "http://example.com",
					Err: &net.OpError{Op: "dial", Err: syscall.ETIMEDOUT},
				},
				nil, // Second attempt succeeds
			},
			responses: []*http.Response{
				nil,               // First attempt fails
				{StatusCode: 200}, // Second attempt succeeds
			},
		}
		mockLogger := &mockLogger{}

		req, _ := http.NewRequest("GET", "http://example.com", nil)
		ctx := context.Background()

		resp, err := DoWithRetry(ctx, mockClient, req, 2, mockLogger)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, 200, resp.StatusCode)
		require.Equal(t, 2, mockClient.callCount)

		// Check that error was logged
		require.True(t, len(mockLogger.errorLogs) > 0)
		require.Contains(t, mockLogger.errorLogs[0], "Retrying")
	})

	t.Run("exhausts all retries and fails", func(t *testing.T) {
		retryableError := &url.Error{
			Op:  "Get",
			URL: "http://example.com",
			Err: &net.OpError{Op: "dial", Err: syscall.ETIMEDOUT},
		}

		mockClient := &mockStandardHTTPClient{
			errors: []error{
				retryableError,
				retryableError,
				retryableError,
			},
		}
		mockLogger := &mockLogger{}

		req, _ := http.NewRequest("GET", "http://example.com", nil)
		ctx := context.Background()

		resp, err := DoWithRetry(ctx, mockClient, req, 3, mockLogger)

		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "failed to make request after 3 attempts")
		require.Equal(t, 3, mockClient.callCount)

		// Check that errors were logged
		require.Equal(t, 3, len(mockLogger.errorLogs))
	})

	t.Run("non-retryable errors return immediately", func(t *testing.T) {
		nonRetryableError := &url.Error{
			Op:  "Get",
			URL: "http://example.com",
			Err: errors.New("invalid URL scheme: []"),
		}

		mockClient := &mockStandardHTTPClient{
			errors: []error{nonRetryableError},
		}
		mockLogger := &mockLogger{}

		req, _ := http.NewRequest("GET", "http://example.com", nil)
		ctx := context.Background()

		resp, err := DoWithRetry(ctx, mockClient, req, 3, mockLogger)

		require.Error(t, err)
		require.Nil(t, resp)
		require.Equal(t, 1, mockClient.callCount) // Should not retry

		// Check that the debug log indicates immediate return
		debugLog := strings.Join(mockLogger.debugLogs, " ")
		require.Contains(t, debugLog, "returning immediately")
	})

	t.Run("context cancellation", func(t *testing.T) {
		mockClient := &mockStandardHTTPClient{
			responses: []*http.Response{{StatusCode: 200}},
		}
		mockLogger := &mockLogger{}

		req, _ := http.NewRequest("GET", "http://example.com", nil)
		ctx, cancel := context.WithCancel(context.Background())

		// Cancel context immediately
		cancel()

		resp, err := DoWithRetry(ctx, mockClient, req, 3, mockLogger)

		require.Error(t, err)
		require.Nil(t, resp)
		require.Equal(t, 0, mockClient.callCount) // Should not make any attempts

		// Check that context cancellation was logged
		debugLog := strings.Join(mockLogger.debugLogs, " ")
		require.Contains(t, debugLog, "Context is done before attempt")
	})

	t.Run("zero retries", func(t *testing.T) {
		mockClient := &mockStandardHTTPClient{
			errors: []error{errors.New("some error")},
		}
		mockLogger := &mockLogger{}

		req, _ := http.NewRequest("GET", "http://example.com", nil)
		ctx := context.Background()

		resp, err := DoWithRetry(ctx, mockClient, req, 0, mockLogger)

		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "failed to make request after 0 attempts")
		require.Equal(t, 0, mockClient.callCount)
	})

	t.Run("negative retries", func(t *testing.T) {
		mockClient := &mockStandardHTTPClient{
			errors: []error{errors.New("some error")},
		}
		mockLogger := &mockLogger{}

		req, _ := http.NewRequest("GET", "http://example.com", nil)
		ctx := context.Background()

		resp, err := DoWithRetry(ctx, mockClient, req, -1, mockLogger)

		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "failed to make request after -1 attempts")
		require.Equal(t, 0, mockClient.callCount)
	})
}

func TestHandleRequestErrorWithLogger(t *testing.T) {
	mockLogger := &mockLogger{}

	tests := []struct {
		name           string
		error          error
		expectedReturn bool
		description    string
	}{
		{
			name:           "BadURL error",
			error:          &url.Error{Op: "Get", URL: "http://example.com", Err: errors.New("invalid URL scheme: []")},
			expectedReturn: true,
			description:    "should return immediately for BadURL errors",
		},
		{
			name:           "ProxyAuthRequired error",
			error:          &url.Error{Op: "Get", URL: "http://example.com", Err: errors.New("Proxy responded with non 200 code: 407 Proxy Authentication Required")},
			expectedReturn: true,
			description:    "should return immediately for proxy auth errors",
		},
		{
			name: "DNSNameNotFound error",
			error: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: &net.DNSError{IsNotFound: true},
			},
			expectedReturn: true,
			description:    "should return immediately for DNS not found errors",
		},
		{
			name:           "retryable timeout error",
			error:          &url.Error{Op: "Get", URL: "http://example.com", Err: &net.OpError{Op: "dial", Err: syscall.ETIMEDOUT}},
			expectedReturn: false,
			description:    "should continue retrying for timeout errors",
		},
		{
			name:           "retryable connection refused error",
			error:          &url.Error{Op: "Get", URL: "http://example.com", Err: &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED}},
			expectedReturn: false,
			description:    "should continue retrying for connection refused errors",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockLogger.reset()
			result := handleRequestError(mockLogger, test.error, 1, 3)

			require.Equal(t, test.expectedReturn, result, test.description)

			// Verify appropriate logs were generated
			if result {
				debugLog := strings.Join(mockLogger.debugLogs, " ")
				require.True(t,
					strings.Contains(debugLog, "returning immediately"),
					"Expected immediate return to be logged")
			} else {
				require.True(t, len(mockLogger.errorLogs) > 0, "Expected error to be logged for retryable errors")
				require.Contains(t, mockLogger.errorLogs[0], "Retrying")
			}
		})
	}
}

func BenchmarkDoWithRetry(b *testing.B) {
	mockClient := &mockStandardHTTPClient{
		responses: make([]*http.Response, b.N),
	}
	for i := 0; i < b.N; i++ {
		mockClient.responses[i] = &http.Response{StatusCode: 200}
	}

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mockClient.callCount = 0 // Reset for each iteration
		_, err := DoWithRetry(ctx, mockClient, req, 1, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}
