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
