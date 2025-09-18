// Package httputils provides utility functions for HTTP operations
package httputils

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/textproto"

	"strings"
	"sync"

	"github.com/andybalholm/brotli"
	fhttp "github.com/bogdanfinn/fhttp"
	"github.com/klauspost/compress/flate"
	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/zlib"
	"github.com/klauspost/compress/zstd"
)

// ResponseWithHeader is a constraint that ensures the response type has a Header and Body field.
// This allows the DecompressResponse function to work with any HTTP response type that provides
// access to headers (e.g., http.Response, custom response types).
type ResponseWithHeader interface {
	Header() map[string][]string
}

var (
	// gzipDecoderPool is a sync.Pool for reusing gzip.Reader instances to reduce memory allocations
	// and improve performance when decompressing multiple gzip responses.
	gzipDecoderPool = sync.Pool{
		New: func() any {
			return new(gzip.Reader)
		},
	}
	// zstdDecoderPool is a sync.Pool for reusing zstd.Decoder instances to reduce memory allocations
	// and improve performance when decompressing multiple zstd responses.
	zstdDecoderPool = sync.Pool{
		New: func() any {
			decoder, err := zstd.NewReader(nil)
			if err != nil {
				panic(err)
			}
			return decoder
		},
	}
)

// DecompressResponse decompresses the response body based on the Content-Encoding header.
// It supports multiple compression formats: gzip, deflate, brotli (br), and zstd.
// If the Content-Encoding is unknown or empty, it returns the body as-is.
func DecompressResponse(headers map[string][]string, body io.ReadCloser) ([]byte, error) {
	// Extract and normalize the Content-Encoding header value
	value := textproto.MIMEHeader(headers).Get("Content-Encoding")
	encoding := strings.ToLower(strings.TrimSpace(value))

	// Route to the appropriate decompression function based on the encoding
	switch encoding {
	case "gzip":
		return GzipDecompress(body)
	case "deflate":
		return DeflateDecompress(body)
	case "br":
		return BrotliDecompress(body)
	case "zstd":
		return ZstdDecompress(body)
	default: // Unknown encoding so just return the body as is
		return io.ReadAll(body)
	}
}

// AnyResponse is a type set of supported response pointer types.
type AnyResponse interface {
	*http.Response | *fhttp.Response
}

// DecompressResponseBody decompresses the response body based on the Content-Encoding header
// and replaces the Response.Body with the decompressed data. This way, consumers can read
// directly from resp.Body without having to deal with compression formats.
//
// TODO: Unit tests
func DecompressResponseBody[T AnyResponse](resp T) error {
	// T is a pointer type per the constraint, so this is safe.
	if any(resp) == nil || getBody(resp) == nil {
		return nil
	}

	// Decompress using your existing helper:
	//   DecompressResponse(headers map[string][]string, body io.ReadCloser) ([]byte, error)
	data, err := DecompressResponse(getHeader(resp), getBody(resp))
	if err != nil {
		return fmt.Errorf("error decompressing response: %w", err)
	}

	// Replace Body
	setBody(resp, io.NopCloser(bytes.NewReader(data)))
	return nil
}

// GzipDecompress decompresses gzip-compressed data using a pooled gzip.Reader for better performance.
// The function automatically closes the input body and manages the lifecycle of the pooled reader.
func GzipDecompress(body io.ReadCloser) ([]byte, error) {
	defer body.Close()

	// Get a gzip reader from the pool and return it when done
	reader := gzipDecoderPool.Get().(*gzip.Reader)
	defer gzipDecoderPool.Put(reader)

	// Reset the reader with the new body data
	if err := reader.Reset(body); err != nil {
		return nil, err
	}
	defer reader.Close()

	// Read all the data from the reader
	return io.ReadAll(reader)
}

// ZstdDecompress decompresses zstd-compressed data using a pooled zstd.Decoder for better performance.
// The function automatically closes the input body and manages the lifecycle of the pooled decoder.
func ZstdDecompress(body io.ReadCloser) ([]byte, error) {
	defer body.Close()

	// Get a zstd decoder from the pool and return it when done
	decoder := zstdDecoderPool.Get().(*zstd.Decoder)
	defer zstdDecoderPool.Put(decoder)

	// Reset the decoder with the new body data
	decoder.Reset(body)
	return io.ReadAll(decoder)
}

// BrotliDecompress decompresses Brotli-compressed data.
func BrotliDecompress(body io.ReadCloser) ([]byte, error) {
	defer body.Close()

	// Create a new Brotli reader and read all data
	brReader := brotli.NewReader(body)
	return io.ReadAll(brReader)
}

// DeflateDecompress decompresses deflate-compressed data with fallback support for different formats.
func DeflateDecompress(body io.ReadCloser) ([]byte, error) {
	defer body.Close()

	// Read the entire compressed body first since we need to try multiple decompression methods
	compressedData, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}

	// First try with zlib (RFC 1950) which is the correct format for HTTP "deflate"
	zlibReader, err := zlib.NewReader(bytes.NewReader(compressedData))
	if err == nil {
		defer zlibReader.Close()
		return io.ReadAll(zlibReader)
	}

	// If zlib fails, try raw deflate as a fallback (RFC 1951)
	// Some servers incorrectly send raw deflate data without the zlib wrapper
	rawReader := flate.NewReader(bytes.NewReader(compressedData))
	defer rawReader.Close()
	return io.ReadAll(rawReader)
}

// getHeader extracts the Header map from the response.
// Since *http.Response and *fhttp.Response both expose a Header field
// of type http.Header, this helper uses a type switch to return it
// in a unified way.
//
// TODO: Unit tests
func getHeader[T AnyResponse](r T) map[string][]string {
	switch v := any(r).(type) {
	case *http.Response:
		return v.Header
	case *fhttp.Response:
		return v.Header
	default:
		// Should never happen because of the AnyResponse constraint
		return nil
	}
}

// getBody extracts the Body (io.ReadCloser) from the response.
// Both response types define Body the same way, but we can’t
// access it directly through a generic type, so we normalize
// access via this helper.
//
// TODO: Unit tests
func getBody[T AnyResponse](r T) io.ReadCloser {
	switch v := any(r).(type) {
	case *http.Response:
		return v.Body
	case *fhttp.Response:
		return v.Body
	default:
		return nil
	}
}

// setBody replaces the Body on the given response with the provided io.ReadCloser.
// This is used after we decompress the original compressed body and need
// to swap in the new decompressed data stream.
//
// TODO: Unit tests
func setBody[T AnyResponse](r T, b io.ReadCloser) {
	switch v := any(r).(type) {
	case *http.Response:
		v.Body = b
	case *fhttp.Response:
		v.Body = b
	}
}
