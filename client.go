package httputils

import "context"

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

// Do is a wrapper around DoWithRetry that provides a simpler interface for making requests.
// It takes a context and a request, and returns a response and an error.
// It will retry the request up to numRetries times in case of network-related errors.
// The function uses intelligent error classification to determine which errors are retryable:
//   - BadURL errors are not retried (immediate failure)
//   - Proxy-related errors are not retried (immediate failure)
func (c *RetryableHTTPClient[T, Rq, Rp]) Do(ctx context.Context, req Rq) (Rp, error) {
	return DoWithRetry(ctx, c.client, req, c.numRetries, c.logger)
}

// DoWithRetry attempts to send an HTTP request using the provided HTTP client.
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
func (c *RetryableHTTPClient[T, Rq, Rp]) DoWithRetry(ctx context.Context, req Rq) (Rp, error) {
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
