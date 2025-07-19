package httputils

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test custom error types
func TestClientErr(t *testing.T) {
	tests := []struct {
		name     string
		err      ClientErr
		expected string
	}{
		{
			name:     "timed out",
			err:      ClientErr{TimedOut: true},
			expected: "timed out",
		},
		{
			name:     "unknown error",
			err:      ClientErr{TimedOut: false},
			expected: "unknown http client error",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.err.Error()
			require.Equal(t, test.expected, result)
		})
	}
}

func TestProxyErr(t *testing.T) {
	tests := []struct {
		name     string
		err      ProxyErr
		expected string
	}{
		{
			name:     "invalid auth",
			err:      ProxyErr{InvalidAuth: true},
			expected: "Proxy auth is incorrect",
		},
		{
			name:     "invalid host",
			err:      ProxyErr{InvalidHost: true},
			expected: "Proxy host is incorrect",
		},
		{
			name:     "invalid port",
			err:      ProxyErr{InvalidPort: true},
			expected: "Proxy port is incorrect",
		},
		{
			name:     "generic proxy error",
			err:      ProxyErr{},
			expected: "Proxy error",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.err.Error()
			require.Equal(t, test.expected, result)
		})
	}
}

func TestRequestErr(t *testing.T) {
	message := "custom request error message"
	err := RequestErr{Message: message}
	result := err.Error()
	require.Equal(t, message, result)
}

// Test ParseError function
func TestParseError(t *testing.T) {
	tests := []struct {
		name         string
		inputError   error
		expectedType string
		description  string
	}{
		{
			name:         "address error with invalid port",
			inputError:   &net.AddrError{Err: "invalid port"},
			expectedType: "*httputils.ProxyErr",
			description:  "should return ProxyErr for invalid port",
		},
		{
			name:         "DNS error not found",
			inputError:   &net.DNSError{IsNotFound: true},
			expectedType: "*httputils.ProxyErr",
			description:  "should return ProxyErr for DNS not found",
		},
		{
			name:         "connection refused",
			inputError:   syscall.ECONNREFUSED,
			expectedType: "*httputils.ProxyErr",
			description:  "should return ProxyErr for connection refused",
		},
		{
			name: "URL timeout error",
			inputError: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: &net.OpError{Op: "dial", Err: &net.DNSError{IsTimeout: true}},
			},
			expectedType: "*httputils.ClientErr",
			description:  "should return ClientErr for timeout",
		},
		{
			name: "EOF error",
			inputError: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: errors.New("EOF"),
			},
			expectedType: "*httputils.RequestErr",
			description:  "should return RequestErr for EOF",
		},
		{
			name: "proxy auth required",
			inputError: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: errors.New("Proxy responded with non 200 code: 407 Proxy Authentication Required"),
			},
			expectedType: "*httputils.ProxyErr",
			description:  "should return ProxyErr for auth required",
		},
		{
			name: "proxy relay offline",
			inputError: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: errors.New("Proxy responded with non 200 code: 502 Proxy Error (The selected relay is offline or busy processing other threads)"),
			},
			expectedType: "*httputils.ProxyErr",
			description:  "should return ProxyErr for relay offline",
		},
		{
			name: "invalid URL scheme",
			inputError: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: errors.New("invalid URL scheme: []"),
			},
			expectedType: "*httputils.ClientErr",
			description:  "should return ClientErr for invalid URL scheme",
		},
		{
			name:         "unknown error",
			inputError:   errors.New("some unknown error"),
			expectedType: "*errors.errorString",
			description:  "should return original error for unknown errors",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := ParseError(test.inputError)
			resultType := fmt.Sprintf("%T", result)

			require.Equal(t, test.expectedType, resultType, test.description)
		})
	}
}

// Test NetworkError constants
func TestNetworkErrorConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant NetworkError
		expected string
	}{
		{"DNS name not found", DNSNameNotFound, "DNS name not found"},
		{"Connection refused", ConnectionRefused, "connection refused"},
		{"Connection timed out", ConnectionTimedOut, "connection timed out"},
		{"Host unreachable", HostUnreachable, "host unreachable"},
		{"Network unreachable", NetworkUnreachable, "network unreachable"},
		{"EOF", EOF, "EOF error"},
		{"Proxy auth required", ProxyAuthRequired, "proxy error - proxy authentication required"},
		{"Proxy relay offline", ProxyRelayOffline, "proxy error - relay is offline or busy processing other threads"},
		{"Bad URL", BadURL, "bad URL"},
		{"Generic error", GenericError, "network error"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := string(test.constant)
			require.Equal(t, test.expected, result)
		})
	}
}

// Test edge cases and error wrapping
func TestErrorWrapping(t *testing.T) {
	t.Run("deeply nested proxy error", func(t *testing.T) {
		baseErr := &ProxyErr{InvalidAuth: true}
		wrappedErr := fmt.Errorf("level 1: %w", baseErr)
		deeplyWrapped := fmt.Errorf("level 2: %w", wrappedErr)

		result := IsProxyError(deeplyWrapped)
		require.True(t, result, "Expected IsProxyError to return true for deeply wrapped ProxyErr")
	})

	t.Run("deeply nested client error", func(t *testing.T) {
		baseErr := &ClientErr{TimedOut: true}
		wrappedErr := fmt.Errorf("level 1: %w", baseErr)
		deeplyWrapped := fmt.Errorf("level 2: %w", wrappedErr)

		result := IsClientError(deeplyWrapped)
		require.True(t, result, "Expected IsClientError to return true for deeply wrapped ClientErr")
	})
}

func TestProxyError(t *testing.T) {
	tests := []struct {
		name         string
		inputError   error
		expectedType string
		expectedErr  *ProxyErr
		description  string
	}{
		{
			name: "proxy with invalid auth",
			inputError: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: errors.New("Proxy responded with non 200 code: 407 Proxy Authentication Required"),
			},
			expectedType: "*httputils.ProxyErr",
			expectedErr:  &ProxyErr{InvalidAuth: true},
			description:  "should return ProxyErr with InvalidAuth for proxy auth error",
		},
		{
			name:         "proxy with invalid host",
			inputError:   &net.DNSError{IsNotFound: true},
			expectedType: "*httputils.ProxyErr",
			expectedErr:  &ProxyErr{InvalidHost: true},
			description:  "should return ProxyErr with InvalidHost for DNS not found error",
		},
		{
			name:         "proxy with invalid port",
			inputError:   &net.AddrError{Err: "invalid port"},
			expectedType: "*httputils.ProxyErr",
			expectedErr:  &ProxyErr{InvalidPort: true},
			description:  "should return ProxyErr with InvalidPort for invalid port error",
		},
		{
			name: "proxy relay offline",
			inputError: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: errors.New("Proxy responded with non 200 code: 502 Proxy Error (The selected relay is offline or busy processing other threads)"),
			},
			expectedType: "*httputils.ProxyErr",
			expectedErr:  &ProxyErr{},
			description:  "should return generic ProxyErr for relay offline error",
		},
		{
			name:         "connection refused",
			inputError:   syscall.ECONNREFUSED,
			expectedType: "*httputils.ProxyErr",
			expectedErr:  &ProxyErr{},
			description:  "should return generic ProxyErr for connection refused error",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := ParseError(test.inputError)
			resultType := fmt.Sprintf("%T", result)

			// Check that the error is of type ProxyErr
			require.Equal(t, test.expectedType, resultType, test.description)

			// Check that IsProxyError returns true
			require.True(t, IsProxyError(result), "Expected IsProxyError to return true for %v", resultType)

			// Check the specific ProxyErr fields
			var proxyErr *ProxyErr
			require.ErrorAs(t, result, &proxyErr, "Failed to convert result to ProxyErr")
			require.Equal(t, test.expectedErr.InvalidAuth, proxyErr.InvalidAuth, "InvalidAuth mismatch")
			require.Equal(t, test.expectedErr.InvalidHost, proxyErr.InvalidHost, "InvalidHost mismatch")
			require.Equal(t, test.expectedErr.InvalidPort, proxyErr.InvalidPort, "InvalidPort mismatch")
		})
	}
}

// Test ClassifyNetworkError function
func TestClassifyNetworkError(t *testing.T) {
	tests := []struct {
		name        string
		inputError  error
		expected    NetworkError
		description string
	}{
		{
			name: "DNS name not found",
			inputError: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: &net.DNSError{IsNotFound: true},
			},
			expected:    DNSNameNotFound,
			description: "should return DNSNameNotFound for DNS not found error",
		},
		{
			name: "DNS error with other error type",
			inputError: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: &net.DNSError{Err: "temporary failure in name resolution"},
			},
			expected:    NetworkError("unknown DNS error: temporary failure in name resolution"),
			description: "should return formatted DNS error for non-IsNotFound DNS errors",
		},
		{
			name: "connection refused syscall error",
			inputError: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED},
			},
			expected:    ConnectionRefused,
			description: "should return ConnectionRefused for ECONNREFUSED syscall error",
		},
		{
			name: "connection refused Windows error code",
			inputError: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: &net.OpError{Op: "dial", Err: syscall.Errno(10061)},
			},
			expected:    ConnectionRefused,
			description: "should return ConnectionRefused for Windows error code 10061",
		},
		{
			name: "connection timeout syscall error",
			inputError: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: &net.OpError{Op: "dial", Err: syscall.ETIMEDOUT},
			},
			expected:    ConnectionTimedOut,
			description: "should return ConnectionTimedOut for ETIMEDOUT syscall error",
		},
		{
			name: "host unreachable syscall error",
			inputError: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: &net.OpError{Op: "dial", Err: syscall.EHOSTUNREACH},
			},
			expected:    HostUnreachable,
			description: "should return HostUnreachable for EHOSTUNREACH syscall error",
		},
		{
			name: "network unreachable syscall error",
			inputError: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: &net.OpError{Op: "dial", Err: syscall.ENETUNREACH},
			},
			expected:    NetworkUnreachable,
			description: "should return NetworkUnreachable for ENETUNREACH syscall error",
		},
		{
			name: "URL timeout error",
			inputError: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: &net.OpError{Op: "dial", Err: &net.DNSError{IsTimeout: true}},
			},
			expected:    NetworkError("unknown DNS error: "),
			description: "should return DNS error for URL timeout with DNS timeout error",
		},
		{
			name: "EOF error",
			inputError: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: errors.New("EOF"),
			},
			expected:    EOF,
			description: "should return EOF for EOF error",
		},
		{
			name: "proxy auth required",
			inputError: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: errors.New("Proxy responded with non 200 code: 407 Proxy Authentication Required"),
			},
			expected:    ProxyAuthRequired,
			description: "should return ProxyAuthRequired for proxy auth error",
		},
		{
			name: "proxy relay offline",
			inputError: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: errors.New("Proxy responded with non 200 code: 502 Proxy Error (The selected relay is offline or busy processing other threads)"),
			},
			expected:    ProxyRelayOffline,
			description: "should return ProxyRelayOffline for proxy relay error",
		},
		{
			name: "invalid URL scheme",
			inputError: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: errors.New("invalid URL scheme: []"),
			},
			expected:    BadURL,
			description: "should return BadURL for invalid URL scheme error",
		},
		{
			name: "generic network error with timeout",
			inputError: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: &mockTimeoutError{timeout: true},
			},
			expected:    ConnectionTimedOut,
			description: "should return ConnectionTimedOut for generic network error with timeout",
		},
		{
			name: "generic network error without timeout",
			inputError: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: &mockTimeoutError{timeout: false},
			},
			expected:    GenericError,
			description: "should return GenericError for generic network error without timeout",
		},
		{
			name:        "unknown error",
			inputError:  errors.New("some completely unknown error"),
			expected:    NetworkError("unknown network error: some completely unknown error"),
			description: "should return formatted unknown error for unrecognized errors",
		},
		{
			name:        "nil error",
			inputError:  nil,
			expected:    NetworkError("unknown network error: %!s(<nil>)"),
			description: "should handle nil error gracefully",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := ClassifyNetworkError(test.inputError)
			require.Equal(t, test.expected, result, test.description)
		})
	}
}

// Test ClassifyNetworkError with deeply wrapped errors
func TestClassifyNetworkErrorWithWrappedErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected NetworkError
	}{
		{
			name: "deeply wrapped DNS error",
			err: fmt.Errorf("level 3: %w",
				fmt.Errorf("level 2: %w",
					fmt.Errorf("level 1: %w",
						&url.Error{
							Op:  "Get",
							URL: "http://example.com",
							Err: &net.DNSError{IsNotFound: true},
						},
					),
				),
			),
			expected: DNSNameNotFound,
		},
		{
			name: "deeply wrapped connection refused",
			err: fmt.Errorf("level 3: %w",
				fmt.Errorf("level 2: %w",
					fmt.Errorf("level 1: %w",
						&url.Error{
							Op:  "Get",
							URL: "http://example.com",
							Err: &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED},
						},
					),
				),
			),
			expected: ConnectionRefused,
		},
		{
			name: "deeply wrapped EOF error",
			err: fmt.Errorf("level 3: %w",
				fmt.Errorf("level 2: %w",
					fmt.Errorf("level 1: %w",
						&url.Error{
							Op:  "Get",
							URL: "http://example.com",
							Err: errors.New("EOF"),
						},
					),
				),
			),
			expected: EOF,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := ClassifyNetworkError(test.err)
			require.Equal(t, test.expected, result)
		})
	}
}

// Test edge cases for ClassifyNetworkError
func TestClassifyNetworkErrorEdgeCases(t *testing.T) {
	t.Run("URL error with empty inner error", func(t *testing.T) {
		urlErr := &url.Error{
			Op:  "Get",
			URL: "http://example.com",
			Err: errors.New(""),
		}
		result := ClassifyNetworkError(urlErr)
		expected := NetworkError("unknown network error: Get \"http://example.com\": ")
		require.Equal(t, expected, result)
	})

	t.Run("unknown syscall error", func(t *testing.T) {
		urlErr := &url.Error{
			Op:  "Get",
			URL: "http://example.com",
			Err: &net.OpError{Op: "dial", Err: syscall.EINVAL},
		}
		result := ClassifyNetworkError(urlErr)
		expected := GenericError
		require.Equal(t, expected, result)
	})
}

func TestConnectionTimedOut(t *testing.T) {

}

// Benchmark tests for performance-critical functions
func BenchmarkParseError(b *testing.B) {
	testErr := &url.Error{
		Op:  "Get",
		URL: "http://example.com",
		Err: errors.New("EOF"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseError(testErr)
	}
}

func BenchmarkClassifyNetworkError(b *testing.B) {
	testErr := &url.Error{
		Op:  "Get",
		URL: "http://example.com",
		Err: &net.DNSError{IsNotFound: true},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ClassifyNetworkError(testErr)
	}
}

// Test helper function for creating mock errors
func createMockURLError(innerErr error) *url.Error {
	return &url.Error{
		Op:  "Get",
		URL: "http://example.com",
		Err: innerErr,
	}
}

// mockTimeoutError implements net.Error for testing
type mockTimeoutError struct {
	timeout bool
}

func (m *mockTimeoutError) Error() string {
	return "mock timeout error"
}

func (m *mockTimeoutError) Timeout() bool {
	return m.timeout
}

func (m *mockTimeoutError) Temporary() bool {
	return false
}
