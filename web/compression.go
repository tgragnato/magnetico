package web

import (
	"bufio"
	"compress/flate"
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
)

type responseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w *responseWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriter) Write(b []byte) (size int, err error) {
	if w.Writer == nil {
		size, err = w.ResponseWriter.Write(b)
	} else {
		size, err = w.Writer.Write(b)
	}
	return size, err
}

func (w *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("underlying ResponseWriter does not support Hijack")
}

func (w *responseWriter) Flush() {
	if w.Writer != nil {
		if flusher, ok := w.Writer.(interface{ Flush() }); ok {
			flusher.Flush()
		}
	}
	if w.ResponseWriter != nil {
		if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
			flusher.Flush()
		}
	}
}

func zstdMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Vary", "Accept-Encoding")
		w.Header().Set("Content-Encoding", "zstd")
		zw, err := zstd.NewWriter(w, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer zw.Close()
		zwr := &responseWriter{Writer: zw, ResponseWriter: w}
		next.ServeHTTP(zwr, r)
	})
}

func brotliMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Vary", "Accept-Encoding")
		w.Header().Set("Content-Encoding", "br")
		br := brotli.NewWriterLevel(w, brotli.BestCompression)
		if br == nil {
			http.Error(w, "brotli.NewWriterLevel is nil", http.StatusInternalServerError)
			return
		}
		defer br.Close()
		brw := &responseWriter{Writer: br, ResponseWriter: w}
		next.ServeHTTP(brw, r)
	})
}

func gzipMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Vary", "Accept-Encoding")
		w.Header().Set("Content-Encoding", "gzip")
		gz, err := gzip.NewWriterLevel(w, gzip.BestCompression)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer gz.Close()
		gzr := &responseWriter{Writer: gz, ResponseWriter: w}
		next.ServeHTTP(gzr, r)
	})
}

func flateMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Vary", "Accept-Encoding")
		w.Header().Set("Content-Encoding", "deflate")
		fl, err := flate.NewWriter(w, flate.BestCompression)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer fl.Close()
		flr := &responseWriter{Writer: fl, ResponseWriter: w}
		next.ServeHTTP(flr, r)
	})
}

func compressMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encodings := strings.Split(r.Header.Get("Accept-Encoding"), ",")
		for index := range encodings {
			encodings[index] = strings.TrimSpace(encodings[index])
			encodings[index] = strings.ToLower(encodings[index])
		}
		sort.Strings(encodings)
		r.Header.Del("Accept-Encoding")
		if i := sort.SearchStrings(encodings, "zstd"); i < len(encodings) && encodings[i] == "zstd" {
			zstdMiddleware(next).ServeHTTP(w, r)
			return
		}
		if i := sort.SearchStrings(encodings, "br"); i < len(encodings) && encodings[i] == "br" {
			brotliMiddleware(next).ServeHTTP(w, r)
			return
		}
		if i := sort.SearchStrings(encodings, "gzip"); i < len(encodings) && encodings[i] == "gzip" {
			gzipMiddleware(next).ServeHTTP(w, r)
			return
		}
		if i := sort.SearchStrings(encodings, "deflate"); i < len(encodings) && encodings[i] == "deflate" {
			flateMiddleware(next).ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}
