package persistence

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func Test_bitmagnet_AddNewTorrent(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	b := &bitmagnet{
		url:        server.URL,
		debug:      true,
		sourceName: "testsource",
		cache:      map[string]time.Time{},
		Mutex:      sync.Mutex{},
	}

	infoHash := []byte("testhash")
	name := "testname"
	files := []File{
		{Size: 100},
		{Size: 200},
	}

	err := b.AddNewTorrent(infoHash, name, files)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	b.Lock()
	_, found := b.cache[string(infoHash)]
	b.Unlock()
	if !found {
		t.Fatalf("expected torrent to be in cache")
	}

	err = b.AddNewTorrent(infoHash, name, files)
	if err == nil || err.Error() != "torrent already exists" {
		t.Fatalf("expected 'torrent already exists' error, got %v", err)
	}
}

func Test_bitmagnet_GetNumberOfTorrents(t *testing.T) {
	t.Parallel()

	b := &bitmagnet{
		url:        "",
		debug:      true,
		sourceName: "testsource",
		cache:      map[string]time.Time{},
		Mutex:      sync.Mutex{},
	}
	got, err := b.GetNumberOfTorrents()
	if err != nil {
		t.Errorf("bitmagnet.GetNumberOfTorrents() error = %v, want nil", err)
	}
	if got != 0 {
		t.Errorf("bitmagnet.GetNumberOfTorrents() = %v, want 0", got)
	}
}

func Test_bitmagnet_QueryTorrents(t *testing.T) {
	t.Parallel()

	b := &bitmagnet{
		url:        "",
		debug:      true,
		sourceName: "testsource",
		cache:      map[string]time.Time{},
		Mutex:      sync.Mutex{},
	}

	got, err := b.QueryTorrents(
		"example query",
		int64(1234567890),
		ByRelevance,
		true,
		uint64(10),
		nil,
		nil,
	)
	if err == nil {
		t.Error("bitmagnet.QueryTorrents() error = nil, want error")
	}
	if got != nil {
		t.Error("bitmagnet.QueryTorrents() != nil, want nil")
	}
}

func Test_bitmagnet_GetTorrent(t *testing.T) {
	t.Parallel()

	b := &bitmagnet{
		url:        "",
		debug:      true,
		sourceName: "testsource",
		cache:      map[string]time.Time{},
		Mutex:      sync.Mutex{},
	}
	got, err := b.GetTorrent([]byte("infoHash"))
	if err == nil {
		t.Error("bitmagnet.GetTorrent() error = nil, want error")
	}
	if got != nil {
		t.Error("bitmagnet.GetTorrent() != nil, want nil")
	}
}

func Test_bitmagnet_GetFiles(t *testing.T) {
	t.Parallel()

	b := &bitmagnet{
		url:        "",
		debug:      true,
		sourceName: "testsource",
		cache:      map[string]time.Time{},
		Mutex:      sync.Mutex{},
	}
	got, err := b.GetFiles([]byte("infoHash"))
	if err == nil {
		t.Error("bitmagnet.GetFiles() error = nil, , wanted error")
	}
	if got != nil {
		t.Errorf("bitmagnet.GetFiles() = %v, want nil", got)
	}
}

func Test_bitmagnet_GetStatistics(t *testing.T) {
	t.Parallel()

	b := &bitmagnet{
		url:        "",
		debug:      true,
		sourceName: "testsource",
		cache:      map[string]time.Time{},
		Mutex:      sync.Mutex{},
	}
	got, err := b.GetStatistics("", 0)
	if err == nil {
		t.Error("bitmagnet.GetStatistics() error = nil, wanted error")
	}
	if got != nil {
		t.Errorf("bitmagnet.GetStatistics() = %v, want nil", got)
	}
}

func Test_bitmagnet_Engine(t *testing.T) {
	t.Parallel()

	b := &bitmagnet{
		url:        "",
		debug:      true,
		sourceName: "testsource",
		cache:      map[string]time.Time{},
		Mutex:      sync.Mutex{},
	}
	if got := b.Engine(); got != Bitmagnet {
		t.Errorf("bitmagnet.Engine() = %v, want %v", got, ZeroMQ)
	}
}

func Test_bitmagnet_cleanup(t *testing.T) {
	t.Parallel()

	b := &bitmagnet{
		url:        "",
		debug:      true,
		sourceName: "testsource",
		cache:      map[string]time.Time{},
		Mutex:      sync.Mutex{},
	}

	// Add expired torrent to cache
	expiredInfoHash := "expiredInfoHash"
	b.cache[expiredInfoHash] = time.Now().Add(-1 * time.Minute)

	// Add valid torrent to cache
	validInfoHash := "validInfoHash"
	b.cache[validInfoHash] = time.Now().Add(10 * time.Minute)

	b.cleanup()

	if _, found := b.cache[expiredInfoHash]; found {
		t.Errorf("bitmagnet.cleanup() did not remove expired torrent")
	}

	if _, found := b.cache[validInfoHash]; !found {
		t.Errorf("bitmagnet.cleanup() removed valid torrent")
	}
}

func Test_bitmagnet_DoesTorrentExist(t *testing.T) {
	t.Parallel()

	b := &bitmagnet{
		url:        "",
		debug:      true,
		sourceName: "testsource",
		cache:      map[string]time.Time{},
		Mutex:      sync.Mutex{},
	}

	infoHash := []byte("testhash")

	exists, err := b.DoesTorrentExist(infoHash)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exists {
		t.Fatalf("expected torrent to not exist")
	}

	b.cache[string(infoHash)] = time.Now().Add(10 * time.Minute)

	exists, err = b.DoesTorrentExist(infoHash)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !exists {
		t.Fatalf("expected torrent to exist")
	}
}
