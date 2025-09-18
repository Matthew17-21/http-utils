// Package httputils provides utility functions for HTTP operations
package httputils

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"strconv"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/flate"
	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/zlib"
	"github.com/klauspost/compress/zstd"

	"strings"
	"sync"
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

// DecompressResponseBody decompresses the response body based on the Content-Encoding header
// and replaces the Response.Body with the decompressed data. This way, consumers can read
// directly from resp.Body without having to deal with compression formats.
func DecompressResponseBody(resp *http.Response) error {
	if resp == nil || resp.Body == nil {
		return nil
	}

	// Decompress the body
	data, err := DecompressResponse(resp.Header, resp.Body)
	if err != nil {
		return fmt.Errorf("error decompressing response: %w", err)
	}

	// Replace the body with the decompressed version
	// Wrap it in an io.NopCloser so it satisfies io.ReadCloser
	resp.Body = io.NopCloser(bytes.NewReader(data))

	// Since it's now decompressed, clear the Content-Encoding header
	resp.Header.Del("Content-Encoding")
	resp.Header.Set("Content-Length", strconv.Itoa(len(data)))

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
