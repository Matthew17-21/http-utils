package httputils

import (
	"context"
	"fmt"
	"net/http"

	tlsHttp "github.com/bogdanfinn/fhttp"
)

// Request is a constraint that allows either standard http.Request or bogdanfinn's Request.
// This enables the RetryableHTTPClient to work with both standard and TLS HTTP clients.
type Request interface {
	*http.Request | *tlsHttp.Request
}

// Response is a constraint that allows either standard http.Response or bogdanfinn's Response.
// This enables the RetryableHTTPClient to work with both standard and TLS HTTP responses.
type Response interface {
	*http.Response | *tlsHttp.Response
}

// Doer is a generic interface that defines the signature for HTTP clients.
// It requires a Do method that takes a request and returns a response with an error.
// This abstraction allows the RetryableHTTPClient to work with any HTTP client implementation.
type Doer[Rq Request, Rp Response] interface {
	Do(Rq) (Rp, error)
}

// RetryableHTTPClient is a generic wrapper around HTTP clients that provides
// retry functionality and logging capabilities.
type RetryableHTTPClient[T Doer[Rq, Rp], Rq Request, Rp Response] struct {
	client     T      // The underlying HTTP client that performs the actual requests
	numRetries int    // Number of retry attempts for failed requests
	logger     Logger // Logger instance for debugging and error reporting
}

// ClientOption is a function type that configures a RetryableHTTPClient.
// This follows the functional options pattern for flexible client configuration.
type ClientOption[T Doer[Rq, Rp], Rq Request, Rp Response] func(c *RetryableHTTPClient[T, Rq, Rp])

// NewRetryableHTTPClient creates a new RetryableHTTPClient with the provided client and options.
func NewRetryableHTTPClient[T Doer[Rq, Rp], Rq Request, Rp Response](
	client T,
	opts ...ClientOption[T, Rq, Rp],
) *RetryableHTTPClient[T, Rq, Rp] {
	c := &RetryableHTTPClient[T, Rq, Rp]{
		client: client,
	}

	// Apply default options first
	for _, opt := range defaultClientOptions[T]() {
		opt(c)
	}

	// Apply custom options
	for _, opt := range opts {
		opt(c)
	}

	return c
}

// defaultClientOptions returns the default configuration options for a RetryableHTTPClient.
// By default, it sets 1 retry attempt and uses a silent logger.
func defaultClientOptions[T Doer[Rq, Rp], Rq Request, Rp Response]() []ClientOption[T, Rq, Rp] {
	return []ClientOption[T, Rq, Rp]{
		WithRetries[T](1),
		WithLogger[T](NewSilentLogger()),
	}
}

// WithRetries creates a ClientOption that sets the number of retry attempts.
// The client will retry failed requests up to this many times before giving up.
func WithRetries[T Doer[Rq, Rp], Rq Request, Rp Response](numRetries int) ClientOption[T, Rq, Rp] {
	return func(c *RetryableHTTPClient[T, Rq, Rp]) {
		c.numRetries = numRetries
	}
}

// WithLogger creates a ClientOption that sets the logger for the client.
// The logger is used for debugging information and error reporting during request attempts.
func WithLogger[T Doer[Rq, Rp], Rq Request, Rp Response](logger Logger) ClientOption[T, Rq, Rp] {
	return func(c *RetryableHTTPClient[T, Rq, Rp]) {
		c.logger = logger
	}
}

// MakeRequest attempts to send an HTTP request using the provided HTTP client.
//
// It will retry the request up to numRetries times in case of network-related errors.
// The function uses intelligent error classification to determine which errors are retryable:
//   - BadURL errors are not retried (immediate failure)
//   - Proxy-related errors are not retried (immediate failure)
//   - Network timeouts, connection errors, and other transient issues are retried
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - req: The HTTP request to send
//
// Returns:
//   - Rp: The HTTP response if successful
//   - error: An error if the request fails after all retry attempts
func (c *RetryableHTTPClient[T, Rq, Rp]) MakeRequest(ctx context.Context, req Rq) (Rp, error) {
	return DoWithRetry(ctx, c.client, req, c.numRetries, c.logger)
}

// handleRequestError processes a request error and determines if the client should
// return immediately or continue with retries.
//
// Returns a bool to indicate if the client should return immediately or continue with retries.
// Returns true if the client should return immediately, false if it should retry.
func (c *RetryableHTTPClient[T, Rq, Rp]) handleRequestError(ctx context.Context, err error, attempt int) bool {
	return handleRequestError(c.logger, err, attempt, c.numRetries)
}

// DoWithRetry is a standalone function that provides retry functionality
// for HTTP requests without requiring the creation of a RetryableHTTPClient.
//
// It will retry the request up to numRetries times in case of network-related errors.
// The function uses intelligent error classification to determine which errors are retryable:
//   - BadURL errors are not retried (immediate failure)
//   - Proxy-related errors are not retried (immediate failure)
//   - Network timeouts, connection errors, and other transient issues are retried
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - client: The HTTP client that implements the Doer interface
//   - req: The HTTP request to send
//   - numRetries: Number of retry attempts (0 means no retries)
//   - logger: Logger instance for debugging and error reporting (optional, uses silent logger if nil)
//
// Returns:
//   - Rp: The HTTP response if successful
//   - error: An error if the request fails after all retry attempts
func DoWithRetry[T Doer[Rq, Rp], Rq Request, Rp Response](
	ctx context.Context,
	client T,
	req Rq,
	numRetries int,
	logger Logger,
) (Rp, error) {
	// Use silent logger if none provided
	if logger == nil {
		logger = NewSilentLogger()
	}

	// Loop through retry attempts
	logger.Debug("Starting DoWithRetry with %d retries configured", numRetries)
	for attempt := 1; attempt <= numRetries; attempt++ {
		// Check if context has been cancelled before attempting the request
		if ctx.Err() != nil {
			logger.Debug("[Attempt %d/%d] Context is done before attempt, returning error", attempt, numRetries)
			return nil, ctx.Err()
		}

		// Attempt to make request using the provided client
		logger.Debug("[Attempt %d/%d] Attempting to make request...", attempt, numRetries)
		resp, err := client.Do(req)
		if err != nil {
			if shouldReturnImmediately := handleRequestError(logger, err, attempt, numRetries); shouldReturnImmediately {
				return nil, err
			}
			continue
		}

		// Request was successful, return the response
		logger.Debug("[Attempt %d/%d] Request successful, returning response...", attempt, numRetries)
		return resp, nil
	}

	// If we've exhausted all retry attempts, return an error
	logger.Debug("All %d attempts failed, returning error", numRetries)
	return nil, fmt.Errorf("failed to make request after %d attempts", numRetries)
}

// handleRequestError is a standalone version of handleRequestError that
// takes a logger as a parameter instead of being a method on RetryableHTTPClient.
//
// Returns true if the client should return immediately, false if it should retry.
func handleRequestError(logger Logger, err error, attempt, numRetries int) bool {
	logger.Debug("[Attempt %d/%d] Request failed with error: %v", attempt, numRetries, err)
	var shouldReturnImmediately bool

	// Classify the error to determine if it's retryable
	networkErr := ClassifyNetworkError(err)
	switch networkErr {
	case BadURL:
		// BadURL errors indicate malformed URLs and should not be retried
		logger.Debug("[Attempt %d/%d] BadURL error detected, returning immediately", attempt, numRetries)
		shouldReturnImmediately = true

	case ProxyAuthRequired, ProxyRelayOffline, DNSNameNotFound:
		// Proxy-related errors indicate configuration issues and should not be retried
		logger.Debug("[Attempt %d/%d] Proxy-related error detected, returning immediately", attempt, numRetries)
		shouldReturnImmediately = true

	default:
		// For other errors (timeouts, connection issues, etc.), log and continue to next attempt
		logger.Error("error while making request: %s. Retrying...", err)
		logger.Debug("[Attempt %d/%d] Network error classified as retryable, continuing to next attempt", attempt, numRetries)
		shouldReturnImmediately = false
	}
	return shouldReturnImmediately
}
