package web

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTorrent(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	if err := torrent().Render(&buffer); err != nil {
		t.Errorf("torrent render: %v", err)
	}
}

func TestTorrentsInfohashHandler(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequest("GET", "/torrents/blablabla", nil)
	if err != nil {
		t.Fatalf("could not create request: %v", err)
	}

	rec := httptest.NewRecorder()
	handler := http.HandlerFunc(torrentsInfohashHandler)
	handler.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status OK; got %v", res.Status)
	}

	contentType := res.Header.Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("expected Content-Type text/html; got %v", contentType)
	}
}
