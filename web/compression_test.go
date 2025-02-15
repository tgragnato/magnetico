package web

import (
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
)

func TestCompressMiddleware(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("test content")); err != nil {
			t.Fatal("Failed to write response:", err)
		}
	})

	tests := []struct {
		name           string
		acceptEncoding string
		wantEncoding   string
		wantError      bool
	}{
		{
			name:           "zstd encoding",
			acceptEncoding: "zstd",
			wantEncoding:   "zstd",
		},
		{
			name:           "brotli encoding",
			acceptEncoding: "br",
			wantEncoding:   "br",
		},
		{
			name:           "gzip encoding",
			acceptEncoding: "gzip",
			wantEncoding:   "gzip",
		},
		{
			name:           "deflate encoding",
			acceptEncoding: "deflate",
			wantEncoding:   "deflate",
		},
		{
			name: "no encoding",
		},
		{
			name:           "unsupported encoding",
			acceptEncoding: "snappy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest("GET", "http://example.com", nil)
			if tt.acceptEncoding != "" {
				req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			}
			rec := httptest.NewRecorder()

			compressMiddleware(handler).ServeHTTP(rec, req)

			var (
				content []byte
				err     error
			)

			switch tt.wantEncoding {

			case "gzip":
				if rec.Header().Get("Content-Encoding") != "gzip" {
					t.Error("Expected Content-Encoding to be gzip")
				}
				var reader *gzip.Reader
				reader, err = gzip.NewReader(rec.Body)
				if err != nil {
					t.Fatal("Failed to create gzip reader:", err)
				}
				defer reader.Close()
				content, err = io.ReadAll(reader)

			case "deflate":
				if rec.Header().Get("Content-Encoding") != "deflate" {
					t.Error("Expected Content-Encoding to be deflate")
				}
				reader := flate.NewReader(rec.Body)
				defer reader.Close()
				content, err = io.ReadAll(reader)

			case "br":
				if rec.Header().Get("Content-Encoding") != "br" {
					t.Error("Expected Content-Encoding to be br")
				}
				content, err = io.ReadAll(brotli.NewReader(rec.Body))

			case "zstd":
				if rec.Header().Get("Content-Encoding") != "zstd" {
					t.Error("Expected Content-Encoding to be ztsd")
				}
				var reader *zstd.Decoder
				reader, err = zstd.NewReader(rec.Body)
				if err != nil {
					t.Fatal("Failed to create zstd reader:", err)
				}
				content, err = io.ReadAll(reader)

			default:
				content, err = io.ReadAll(rec.Body)
			}

			if err != nil && !tt.wantError {
				t.Fatal("Failed to read content:", err)
			}

			if string(content) != "test content" {
				t.Errorf("Expected 'test content' but got '%s'", string(content))
			}
		})
	}
}

func TestResponseWriter_Flush(t *testing.T) {
	t.Parallel()

	t.Run("flush both writer and response writer", func(t *testing.T) {
		t.Parallel()

		rec := httptest.NewRecorder()
		gz := gzip.NewWriter(rec)
		w := &responseWriter{
			Writer:         gz,
			ResponseWriter: rec,
		}

		w.Flush()

		if rec.Flushed != true {
			t.Error("Expected ResponseWriter to be flushed")
		}
	})

	t.Run("flush with nil writer", func(t *testing.T) {
		t.Parallel()

		rec := httptest.NewRecorder()
		w := &responseWriter{
			ResponseWriter: rec,
		}

		w.Flush()

		if rec.Flushed != true {
			t.Error("Expected ResponseWriter to be flushed")
		}
	})

	t.Run("flush with nil response writer", func(t *testing.T) {
		t.Parallel()

		var buf strings.Builder
		w := &responseWriter{
			Writer: &buf,
		}

		w.Flush()
	})
}
