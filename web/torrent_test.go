package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tgragnato.it/magnetico/v2/persistence"
	"tgragnato.it/magnetico/v2/types/infohash"
)

func TestTorrent(t *testing.T) {
	t.Parallel()

	infohashV1 := "abcdef1234567890abcdef1234567890abcdef12"
	torrentMetadata := persistence.TorrentMetadata{
		Name:         "Test Torrent",
		Size:         12345678,
		DiscoveredOn: 1710000000,
		NFiles:       2,
	}
	files := []persistence.File{
		{Path: "file1.txt", Size: 1000},
		{Path: "file2.bin", Size: 2000},
	}

	node := torrent(infohashV1, torrentMetadata, files)
	rec := httptest.NewRecorder()
	if err := node.Render(rec); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	html := rec.Body.String()

	if !strings.Contains(html, "Test Torrent") {
		t.Errorf("Torrent name not found in HTML")
	}
	if !strings.Contains(html, "file1.txt") || !strings.Contains(html, "file2.bin") {
		t.Errorf("File names not found in HTML")
	}
}

func TestTorrentsInfohashHandler(t *testing.T) {
	t.Parallel()

	initDb()

	req, err := http.NewRequest("GET", "/torrents/blablabla", nil)
	if err != nil {
		t.Fatalf("could not create request: %v", err)
	}

	rec := httptest.NewRecorder()
	handler := http.HandlerFunc(torrentsInfohashHandler)
	handler.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Errorf("expected status Not Found; got %v", res.Status)
	}
	if contentType := res.Header.Get("Content-Type"); contentType != ContentTypeText {
		t.Errorf("expected Content-Type text/plain; got %v", contentType)
	}

	infohashV1 := "1234567890123456789012345678901234567890"
	req, err = http.NewRequest("GET", "/torrents/"+infohashV1, nil)
	if err != nil {
		t.Fatalf("could not create request: %v", err)
	}

	ctx := context.WithValue(req.Context(), InfohashKey, infohash.FromHexString(infohashV1).Bytes())
	req = req.WithContext(ctx)

	rec = httptest.NewRecorder()
	torrentsInfohashHandler(rec, req)
	res = rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Errorf("expected status Not Found; got %v", res.StatusCode)
	}

	if contentType := res.Header.Get("Content-Type"); contentType != ContentTypeText {
		t.Errorf("expected Content-Type text/plain; got %v", contentType)
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
