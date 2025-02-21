package web

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"tgragnato.it/magnetico/types/infohash"
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

	if contentType := res.Header.Get("Content-Type"); contentType != ContentTypeHtml {
		t.Errorf("expected Content-Type text/html; got %v", contentType)
	}
}

func TestApiTorrent(t *testing.T) {
	t.Parallel()

	initDb()

	infohashV1 := "1234567890123456789012345678901234567890"
	req, err := http.NewRequest("GET", "/api/v0.1/torrents/"+infohashV1, nil)
	if err != nil {
		t.Fatalf("could not create request: %v", err)
	}

	ctx := context.WithValue(req.Context(), InfohashKey, infohash.FromHexString(infohashV1).Bytes())
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	apiTorrent(rec, req)
	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Errorf("expected status %v; got %v", http.StatusNotFound, res.StatusCode)
	}

	if rec.Body.String() != "Not found\n" {
		t.Errorf("expected Not found in body; got %s", rec.Body.String())
	}
}

func TestApiFileList(t *testing.T) {
	t.Parallel()

	initDb()

	infohashV1 := "1234567890123456789012345678901234567890"
	req, err := http.NewRequest("GET", "/api/v0.1/files/"+infohashV1, nil)
	if err != nil {
		t.Fatalf("could not create request: %v", err)
	}

	ctx := context.WithValue(req.Context(), InfohashKey, infohash.FromHexString(infohashV1).Bytes())
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	apiFileList(rec, req)
	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Errorf("expected status %v; got %v", http.StatusNotFound, res.StatusCode)
	}

	if rec.Body.String() != "Not found\n" {
		t.Errorf("expected Not found in body; got %s", rec.Body.String())
	}
}
