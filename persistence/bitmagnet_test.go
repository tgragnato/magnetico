package persistence

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"
)

func Test_bitmagnet_DHTRoutingTable(t *testing.T) {
	t.Parallel()

	_, filter, err := net.ParseCIDR("127.0.0.0/8")
	if err != nil {
		t.Fatal(err)
	}
	db := &bitmagnet{}
	if err := db.ConfigureDHTRoutingTable(2, []net.IPNet{*filter}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	first := net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 6881}
	second := net.UDPAddr{IP: net.IPv4(127, 0, 0, 2), Port: 6882}
	if err := db.AddDHTNodes(ctx, []net.UDPAddr{first, first, second, {IP: net.IPv4(10, 0, 0, 1), Port: 6883}}); err != nil {
		t.Fatal(err)
	}
	if empty, err := db.IsDHTRoutingEmpty(ctx); err != nil || empty {
		t.Fatalf("IsEmpty() = (%v, %v), want (false, nil)", empty, err)
	}

	dumped, err := db.DumpDHTNodes(ctx, true)
	if err != nil || len(dumped) != 2 {
		t.Fatalf("Dump() returned %d nodes and error %v, want 2 nodes and nil", len(dumped), err)
	}
	if dumped, err := db.DumpDHTNodes(ctx, false); err != nil || len(dumped) != 0 {
		t.Fatalf("Dump(IPv6) returned %d nodes and error %v, want 0 nodes and nil", len(dumped), err)
	}

	got, err := db.GetDHTNodes(ctx)
	if err != nil || len(got) != 2 {
		t.Fatalf("GetNodes() returned %d nodes and error %v, want 2 nodes and nil", len(got), err)
	}
	got, err = db.GetDHTNodes(ctx)
	if err != nil || len(got) != 1 || !got[0].IP.Equal(first.IP) || got[0].Port != first.Port {
		t.Fatalf("second GetNodes() = (%v, %v), want only %v", got, err, first)
	}
	if empty, err := db.IsDHTRoutingEmpty(ctx); err != nil || !empty {
		t.Fatalf("IsEmpty() after consumption = (%v, %v), want (true, nil)", empty, err)
	}

	initialHashes := [][20]byte{{1}, {2}}
	if err := db.AddDHTHashes(ctx, initialHashes); err != nil {
		t.Fatal(err)
	}
	if err := db.AddDHTHashes(ctx, [][20]byte{{3}}); err != nil {
		t.Fatal(err)
	}
	gotHashes, err := db.GetDHTHashes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if gotHashes[0] != [20]byte{3} || gotHashes[1] != [20]byte{2} {
		t.Fatalf("GetHashes() = %v, want first hash replaced and second retained", gotHashes[:2])
	}
}

func Test_bitmagnet_DHTOutbox(t *testing.T) {
	t.Parallel()
	db := &bitmagnet{}
	message := DHTOutboundMessage{
		Destination: net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 6881},
		Payload:     []byte("payload"),
		QueuedAt:    1,
	}
	if err := db.EnqueueDHTMessage(context.Background(), message); err != nil {
		t.Fatal(err)
	}
	if err := db.EnqueueDHTMessage(context.Background(), DHTOutboundMessage{
		Destination: message.Destination,
		Payload:     []byte("next"),
		QueuedAt:    2,
	}); err != nil {
		t.Fatal(err)
	}
	queued, found, err := db.NextDHTMessage(context.Background())
	if err != nil || !found || queued.ID == "" || string(queued.Payload) != "payload" {
		t.Fatalf("NextDHTMessage() = (%+v, %v, %v), want queued payload", queued, found, err)
	}
	if err := db.ReleaseDHTMessage(context.Background(), queued.ID); err != nil {
		t.Fatal(err)
	}
	next, found, err := db.NextDHTMessage(context.Background())
	if err != nil || !found || string(next.Payload) != "next" {
		t.Fatalf("NextDHTMessage() after release = (%+v, %v, %v), want next message", next, found, err)
	}
	if err := db.AckDHTMessage(context.Background(), next.ID); err != nil {
		t.Fatal(err)
	}
	retried, found, err := db.NextDHTMessage(context.Background())
	if err != nil || !found || retried.ID != queued.ID {
		t.Fatalf("NextDHTMessage() after next ack = (%+v, %v, %v), want released message", retried, found, err)
	}
	if err := db.AckDHTMessage(context.Background(), retried.ID); err != nil {
		t.Fatal(err)
	}
	if _, found, err := db.NextDHTMessage(context.Background()); err != nil || found {
		t.Fatalf("NextDHTMessage() after ack found=%v err=%v, want no message", found, err)
	}
}

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

func Test_makeBitmagnet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		urlStr     string
		wantErr    bool
		wantDebug  bool
		wantSource string
		wantUrl    string
	}{
		{
			name:       "valid URL with debug and source",
			urlStr:     "bitmagnet://example.com?debug=true&source=testsource",
			wantErr:    false,
			wantDebug:  true,
			wantSource: "testsource",
			wantUrl:    "http://example.com",
		},
		{
			name:       "valid URL without debug and source",
			urlStr:     "bitmagnet://example.com",
			wantErr:    false,
			wantDebug:  false,
			wantSource: "magnetico",
			wantUrl:    "http://example.com",
		},
		{
			name:       "invalid URL",
			urlStr:     "://example.com",
			wantErr:    true,
			wantDebug:  false,
			wantSource: "",
			wantUrl:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url_, err := url.Parse(tt.urlStr)
			if err != nil {
				if !tt.wantErr {
					t.Fatalf("url.Parse() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			got, err := makeBitmagnet(url_)
			if (err != nil) != tt.wantErr {
				t.Fatalf("makeBitmagnet() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}

			b := got.(*bitmagnet)
			if b.debug != tt.wantDebug {
				t.Errorf("bitmagnet.debug = %v, want %v", b.debug, tt.wantDebug)
			}
			if b.sourceName != tt.wantSource {
				t.Errorf("bitmagnet.sourceName = %v, want %v", b.sourceName, tt.wantSource)
			}
			if b.url != tt.wantUrl {
				t.Errorf("bitmagnet.url = %v, want %v", b.url, tt.wantUrl)
			}
		})
	}
}

func Test_bitmagnet_Export(t *testing.T) {
	t.Parallel()

	b := &bitmagnet{
		url:        "",
		debug:      true,
		sourceName: "testsource",
		cache:      map[string]time.Time{},
		Mutex:      sync.Mutex{},
	}
	got, err := b.Export()
	if err == nil {
		t.Error("bitmagnet.Export() error = nil, wanted error")
	}
	if got != nil {
		t.Errorf("bitmagnet.Export() = %v, want nil", got)
	}
}
