# http-utils

Helper functions for Go HTTP clients.

## Features

- **Generic HTTP Client Support**: Works with both standard `net/http` and [`bogdanfinn's`](https://github.com/bogdanfinn/tls-client/) http client
- **Intelligent Retry Logic**: Automatically retries on network errors while avoiding retries on configuration issues
- **Flexible Logging**: Built-in logging support with multiple logger implementations
- **Context Support**: Full context cancellation and timeout support
- **Standalone Functions**: Use retry functionality without creating client instances
- **Response Decompression**: Automatic decompression of gzip, deflate, brotli, and zstd compressed responses

## Quick Start

### Examples

#### Using the standalone function

```go
package main

import (
    "context"
    "net/http"
    "github.com/Matthew17-21/http-utils"
)

func main() {
    // Create a standard HTTP client
    httpClient := &http.Client{}
    
    // Create a request
    req, _ := http.NewRequest("GET", "https://api.example.com/data", nil)
    ctx := context.Background()
    
    // Use the standalone function with retry capability
    resp, err := httputils.DoWithRetry(ctx, httpClient, req, 3, nil)
    if err != nil {
        // Handle error
    }
    defer resp.Body.Close()
    
    // Process response
}
```

#### Using the RetryableHTTPClient

```go
package main

import (
    "context"
    "net/http"
    "github.com/Matthew17-21/http-utils"
)

func main() {
    // Create a standard HTTP client
    httpClient := &http.Client{}
    
    // Create a retryable client with options
    retryableClient := httputils.NewRetryableHTTPClient(
		client,
		httputils.WithRetries[*http.Client](3),
		httputils.WithLogger[*http.Client](httputils.NewLogger()),
	)
    
    // Create a request
    req, _ := http.NewRequest("GET", "https://api.example.com/data", nil)
    ctx := context.Background()
    
    // Make request with automatic retries
    resp, err := retryableClient.MakeRequest(ctx, req)
    if err != nil {
        // Handle error
    }
    defer resp.Body.Close()
    
    // Process response
}
```

#### With Custom Logger

```go
logger := httputils.NewLogger()
resp, err := httputils.DoWithRetry(ctx, httpClient, req, 3, logger)
```

#### With Context Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

resp, err := httputils.DoWithRetry(ctx, httpClient, req, 3, nil)
```

#### With TLS HTTP Client

```go
import tlsHttp "github.com/bogdanfinn/fhttp"

tlsClient := &tlsHttp.Client{}
tlsReq, _ := tlsHttp.NewRequest("GET", "https://api.example.com/data", nil)

resp, err := httputils.DoWithRetry(ctx, tlsClient, tlsReq, 3, nil)
```

#### Decompressing Different Content Types

The library supports automatic decompression of various compression formats. Here are examples for each supported format:

##### Gzip Decompression

```go
// Request gzip-compressed content
req, _ := http.NewRequest("GET", "https://api.example.com/gzipped-data", nil)
req.Header.Set("Accept-Encoding", "gzip")

resp, err := httputils.DoWithRetry(ctx, httpClient, req, 3, nil)
if err != nil {
    return err
}
defer resp.Body.Close()

// Decompress the response
decompressed, err := httputils.DecompressResponse(resp.Header, resp.Body)
if err != nil {
    return err
}
fmt.Println("Decompressed content:", string(decompressed))
```

##### Brotli Decompression

```go
// Request brotli-compressed content
req, _ := http.NewRequest("GET", "https://httpbin.org/brotli", nil)
req.Header.Set("Accept-Encoding", "br")

resp, err := httputils.DoWithRetry(ctx, httpClient, req, 3, nil)
if err != nil {
    return err
}
defer resp.Body.Close()

// Decompress the response
decompressed, err := httputils.DecompressResponse(resp.Header, resp.Body)
if err != nil {
    return err
}
fmt.Println("Decompressed content:", string(decompressed))
```

##### Zstd Decompression

```go
// Request zstd-compressed content
req, _ := http.NewRequest("GET", "https://api.example.com/zstd-data", nil)
req.Header.Set("Accept-Encoding", "zstd")

resp, err := httputils.DoWithRetry(ctx, httpClient, req, 3, nil)
if err != nil {
    return err
}
defer resp.Body.Close()

// Decompress the response
decompressed, err := httputils.DecompressResponse(resp.Header, resp.Body)
if err != nil {
    return err
}
fmt.Println("Decompressed content:", string(decompressed))
```

##### Deflate Decompression

```go
// Request deflate-compressed content
req, _ := http.NewRequest("GET", "https://api.example.com/deflate-data", nil)
req.Header.Set("Accept-Encoding", "deflate")

resp, err := httputils.DoWithRetry(ctx, httpClient, req, 3, nil)
if err != nil {
    return err
}
defer resp.Body.Close()

// Decompress the response
decompressed, err := httputils.DecompressResponse(resp.Header, resp.Body)
if err != nil {
    return err
}
fmt.Println("Decompressed content:", string(decompressed))
```

##### Automatic Decompression with Multiple Formats

```go
// Request any supported compression format
req, _ := http.NewRequest("GET", "https://api.example.com/compressed-data", nil)
req.Header.Set("Accept-Encoding", "br, gzip, deflate, zstd")

resp, err := httputils.DoWithRetry(ctx, httpClient, req, 3, nil)
if err != nil {
    return err
}
defer resp.Body.Close()

// The library automatically detects and decompresses based on Content-Encoding header
decompressed, err := httputils.DecompressResponse(resp.Header, resp.Body)
if err != nil {
    return err
}
fmt.Println("Decompressed content:", string(decompressed))
```

**Note:** When using `bogdanfinn/fhttp` client, some decompression (such as brotli) is handled automatically by the client itself, so you don't need to manually decompress responses.

## Installation

```bash
go get github.com/Matthew17-21/http-utils
```

## Why?

Sometimes I use different HTTP clients and wanted to make it easier to work with them consistently. This library provides a unified interface for retry logic, logging, and response handling across different HTTP client implementations.

## Acknowledgments

The decompression functionality in this library includes code adapted from the [Hyper Solutions Go SDK](https://github.com/Hyper-Solutions/hyper-sdk-go).
