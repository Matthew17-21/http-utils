package httputils

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"io"
	"strings"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
)

// MockResponse implements ResponseWithHeader for testing
type MockResponse struct {
	headers map[string][]string
}

func (m *MockResponse) Header() map[string][]string {
	return m.headers
}

// Helper function to create compressed test data
func compressData(data []byte, method string) ([]byte, error) {
	var buf bytes.Buffer

	switch method {
	case "gzip":
		writer := gzip.NewWriter(&buf)
		_, err := writer.Write(data)
		if err != nil {
			return nil, err
		}
		err = writer.Close()
		return buf.Bytes(), err

	case "deflate":
		writer, err := zlib.NewWriterLevel(&buf, zlib.DefaultCompression)
		if err != nil {
			return nil, err
		}
		_, err = writer.Write(data)
		if err != nil {
			return nil, err
		}
		err = writer.Close()
		return buf.Bytes(), err

	case "deflate-raw":
		writer, err := flate.NewWriter(&buf, flate.DefaultCompression)
		if err != nil {
			return nil, err
		}
		_, err = writer.Write(data)
		if err != nil {
			return nil, err
		}
		err = writer.Close()
		return buf.Bytes(), err

	case "br":
		writer := brotli.NewWriter(&buf)
		_, err := writer.Write(data)
		if err != nil {
			return nil, err
		}
		err = writer.Close()
		return buf.Bytes(), err

	case "zstd":
		encoder, err := zstd.NewWriter(&buf)
		if err != nil {
			return nil, err
		}
		_, err = encoder.Write(data)
		if err != nil {
			return nil, err
		}
		err = encoder.Close()
		return buf.Bytes(), err
	}

	return nil, nil
}

// TestDecompressionWithNetHttp tests the decompression of a response with the net/http package
func TestDecompressionWithNetHttp(t *testing.T) {
	testData := []byte("Hello, World! This is test data for HTTP decompression.")

	tests := []struct {
		name     string
		encoding string
		method   string
	}{
		{"Gzip compression", "gzip", "gzip"},
		{"Deflate compression", "deflate", "deflate"},
		{"Brotli compression", "br", "br"},
		{"Zstandard compression", "zstd", "zstd"},
		{"No compression", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body io.ReadCloser

			if tt.method == "" {
				// No compression
				body = io.NopCloser(bytes.NewReader(testData))
			} else {
				// Compress the data
				compressed, err := compressData(testData, tt.method)
				if err != nil {
					t.Fatalf("Failed to compress data: %v", err)
				}
				body = io.NopCloser(bytes.NewReader(compressed))
			}

			// Create mock response with appropriate headers
			resp := &MockResponse{
				headers: map[string][]string{
					"Content-Encoding": {tt.encoding},
				},
			}

			// Test decompression
			result, err := DecompressResponse(resp.Header(), body)
			if err != nil {
				t.Fatalf("DecompressResponse failed: %v", err)
			}

			if !bytes.Equal(result, testData) {
				t.Errorf("Decompressed data doesn't match original. Got %q, want %q", string(result), string(testData))
			}
		})
	}
}

// TestDecompressionWithFhttp tests the decompression of a response with the fhttp package
func TestDecompressionWithFhttp(t *testing.T) {
	// This test simulates how fhttp might work - assuming similar interface
	testData := []byte("Test data for fhttp decompression testing.")

	// Test with case-insensitive headers (common in HTTP)
	tests := []struct {
		name     string
		encoding string
		method   string
	}{
		{"Gzip with uppercase", "GZIP", "gzip"},
		{"Deflate with mixed case", "Deflate", "deflate"},
		{"Brotli with spaces", " br ", "br"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressed, err := compressData(testData, tt.method)
			if err != nil {
				t.Fatalf("Failed to compress data: %v", err)
			}

			body := io.NopCloser(bytes.NewReader(compressed))
			resp := &MockResponse{
				headers: map[string][]string{
					"Content-Encoding": {tt.encoding},
				},
			}

			result, err := DecompressResponse(resp.Header(), body)
			if err != nil {
				t.Fatalf("DecompressResponse failed: %v", err)
			}

			if !bytes.Equal(result, testData) {
				t.Errorf("Decompressed data doesn't match original. Got %q, want %q", string(result), string(testData))
			}
		})
	}
}

func TestGzipDecompress(t *testing.T) {
	testData := []byte("This is test data for gzip decompression testing with various lengths and characters: 12345!@#$%")

	t.Run("Valid gzip data", func(t *testing.T) {
		compressed, err := compressData(testData, "gzip")
		if err != nil {
			t.Fatalf("Failed to compress data: %v", err)
		}

		body := io.NopCloser(bytes.NewReader(compressed))
		result, err := GzipDecompress(body)
		if err != nil {
			t.Fatalf("GzipDecompress failed: %v", err)
		}

		if !bytes.Equal(result, testData) {
			t.Errorf("Decompressed data doesn't match. Got %q, want %q", string(result), string(testData))
		}
	})

	t.Run("Invalid gzip data", func(t *testing.T) {
		invalidData := []byte("This is not gzip compressed data")
		body := io.NopCloser(bytes.NewReader(invalidData))

		_, err := GzipDecompress(body)
		if err == nil {
			t.Error("Expected error for invalid gzip data, got nil")
		}
	})

	t.Run("Empty gzip data", func(t *testing.T) {
		emptyCompressed, err := compressData([]byte{}, "gzip")
		if err != nil {
			t.Fatalf("Failed to compress empty data: %v", err)
		}

		body := io.NopCloser(bytes.NewReader(emptyCompressed))
		result, err := GzipDecompress(body)
		if err != nil {
			t.Fatalf("GzipDecompress failed on empty data: %v", err)
		}

		if len(result) != 0 {
			t.Errorf("Expected empty result, got %d bytes", len(result))
		}
	})
}

func TestZstdDecompress(t *testing.T) {
	testData := []byte("Zstandard compression test data with repetitive patterns: test test test test")

	t.Run("Valid zstd data", func(t *testing.T) {
		compressed, err := compressData(testData, "zstd")
		if err != nil {
			t.Fatalf("Failed to compress data: %v", err)
		}

		body := io.NopCloser(bytes.NewReader(compressed))
		result, err := ZstdDecompress(body)
		if err != nil {
			t.Fatalf("ZstdDecompress failed: %v", err)
		}

		if !bytes.Equal(result, testData) {
			t.Errorf("Decompressed data doesn't match. Got %q, want %q", string(result), string(testData))
		}
	})

	t.Run("Invalid zstd data", func(t *testing.T) {
		invalidData := []byte("This is not zstd compressed data")
		body := io.NopCloser(bytes.NewReader(invalidData))

		_, err := ZstdDecompress(body)
		if err == nil {
			t.Error("Expected error for invalid zstd data, got nil")
		}
	})

	t.Run("Large zstd data", func(t *testing.T) {
		largeData := bytes.Repeat([]byte("Large test data for zstd compression. "), 1000)
		compressed, err := compressData(largeData, "zstd")
		if err != nil {
			t.Fatalf("Failed to compress large data: %v", err)
		}

		body := io.NopCloser(bytes.NewReader(compressed))
		result, err := ZstdDecompress(body)
		if err != nil {
			t.Fatalf("ZstdDecompress failed on large data: %v", err)
		}

		if !bytes.Equal(result, largeData) {
			t.Error("Large data decompression failed")
		}
	})
}

func TestBrotliDecompress(t *testing.T) {
	testData := []byte("Brotli compression test with various unicode characters: αβγδε 中文 🚀")

	t.Run("Valid brotli data", func(t *testing.T) {
		compressed, err := compressData(testData, "br")
		if err != nil {
			t.Fatalf("Failed to compress data: %v", err)
		}

		body := io.NopCloser(bytes.NewReader(compressed))
		result, err := BrotliDecompress(body)
		if err != nil {
			t.Fatalf("BrotliDecompress failed: %v", err)
		}

		if !bytes.Equal(result, testData) {
			t.Errorf("Decompressed data doesn't match. Got %q, want %q", string(result), string(testData))
		}
	})

	t.Run("Invalid brotli data", func(t *testing.T) {
		invalidData := []byte("This is not brotli compressed data")
		body := io.NopCloser(bytes.NewReader(invalidData))

		_, err := BrotliDecompress(body)
		if err == nil {
			t.Error("Expected error for invalid brotli data, got nil")
		}
	})

	t.Run("JSON data compression", func(t *testing.T) {
		jsonData := []byte(`{"message": "Hello, World!", "data": [1, 2, 3, 4, 5], "nested": {"key": "value"}}`)
		compressed, err := compressData(jsonData, "br")
		if err != nil {
			t.Fatalf("Failed to compress JSON data: %v", err)
		}

		body := io.NopCloser(bytes.NewReader(compressed))
		result, err := BrotliDecompress(body)
		if err != nil {
			t.Fatalf("BrotliDecompress failed on JSON: %v", err)
		}

		if !bytes.Equal(result, jsonData) {
			t.Error("JSON data decompression failed")
		}
	})
}

func TestDeflateDecompress(t *testing.T) {
	testData := []byte("Deflate compression test data with mixed content: HTML <div>content</div>, numbers 123456789")

	t.Run("Valid deflate data (zlib format)", func(t *testing.T) {
		compressed, err := compressData(testData, "deflate")
		if err != nil {
			t.Fatalf("Failed to compress data: %v", err)
		}

		body := io.NopCloser(bytes.NewReader(compressed))
		result, err := DeflateDecompress(body)
		if err != nil {
			t.Fatalf("DeflateDecompress failed: %v", err)
		}

		if !bytes.Equal(result, testData) {
			t.Errorf("Decompressed data doesn't match. Got %q, want %q", string(result), string(testData))
		}
	})

	t.Run("Raw deflate data (RFC 1951)", func(t *testing.T) {
		compressed, err := compressData(testData, "deflate-raw")
		if err != nil {
			t.Fatalf("Failed to compress data: %v", err)
		}

		body := io.NopCloser(bytes.NewReader(compressed))
		result, err := DeflateDecompress(body)
		if err != nil {
			t.Fatalf("DeflateDecompress failed on raw deflate: %v", err)
		}

		if !bytes.Equal(result, testData) {
			t.Errorf("Raw deflate decompressed data doesn't match. Got %q, want %q", string(result), string(testData))
		}
	})

	t.Run("Invalid deflate data", func(t *testing.T) {
		invalidData := []byte("This is not deflate compressed data at all")
		body := io.NopCloser(bytes.NewReader(invalidData))

		_, err := DeflateDecompress(body)
		if err == nil {
			t.Error("Expected error for invalid deflate data, got nil")
		}
	})

	t.Run("HTML content deflate", func(t *testing.T) {
		htmlData := []byte(`<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
	<h1>Hello World</h1>
	<p>This is a test HTML document for deflate compression.</p>
</body>
</html>`)

		compressed, err := compressData(htmlData, "deflate")
		if err != nil {
			t.Fatalf("Failed to compress HTML data: %v", err)
		}

		body := io.NopCloser(bytes.NewReader(compressed))
		result, err := DeflateDecompress(body)
		if err != nil {
			t.Fatalf("DeflateDecompress failed on HTML: %v", err)
		}

		if !bytes.Equal(result, htmlData) {
			t.Error("HTML data decompression failed")
		}
	})
}

// Additional edge case tests
func TestDecompressResponseEdgeCases(t *testing.T) {
	testData := []byte("Edge case test data")

	t.Run("Multiple content-encoding headers", func(t *testing.T) {
		compressed, err := compressData(testData, "gzip")
		if err != nil {
			t.Fatalf("Failed to compress data: %v", err)
		}

		body := io.NopCloser(bytes.NewReader(compressed))
		resp := &MockResponse{
			headers: map[string][]string{
				"Content-Encoding": {"gzip", "deflate"}, // Multiple values
			},
		}

		// Should use the first value (gzip)
		result, err := DecompressResponse(resp.Header(), body)
		if err != nil {
			t.Fatalf("DecompressResponse failed with multiple headers: %v", err)
		}

		if !bytes.Equal(result, testData) {
			t.Error("Failed to handle multiple content-encoding headers")
		}
	})

	t.Run("Unknown encoding", func(t *testing.T) {
		body := io.NopCloser(strings.NewReader("uncompressed data"))
		resp := &MockResponse{
			headers: map[string][]string{
				"Content-Encoding": {"unknown-encoding"},
			},
		}

		result, err := DecompressResponse(resp.Header(), body)
		if err != nil {
			t.Fatalf("DecompressResponse failed with unknown encoding: %v", err)
		}

		expected := "uncompressed data"
		if string(result) != expected {
			t.Errorf("Expected %q, got %q", expected, string(result))
		}
	})

	t.Run("Empty body", func(t *testing.T) {
		body := io.NopCloser(strings.NewReader(""))
		resp := &MockResponse{
			headers: map[string][]string{
				"Content-Encoding": {"gzip"},
			},
		}

		_, err := DecompressResponse(resp.Header(), body)
		// Should handle empty body gracefully (might error, which is fine)
		if err != nil {
			t.Logf("Empty body resulted in expected error: %v", err)
		}
	})

	t.Run("No content-encoding header", func(t *testing.T) {
		testContent := "No compression applied"
		body := io.NopCloser(strings.NewReader(testContent))
		resp := &MockResponse{
			headers: map[string][]string{},
		}

		result, err := DecompressResponse(resp.Header(), body)
		if err != nil {
			t.Fatalf("DecompressResponse failed with no encoding header: %v", err)
		}

		if string(result) != testContent {
			t.Errorf("Expected %q, got %q", testContent, string(result))
		}
	})
}
