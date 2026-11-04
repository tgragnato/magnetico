//go:build cgo

package persistence

import (
	"context"
	"net"
	"net/url"
	"reflect"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func newDb(t *testing.T) Database {
	t.Helper()
	db, err := makeSqlite3Database(&url.URL{
		Scheme:   "sqlite3",
		Path:     strings.ReplaceAll(t.Name(), "/", "_"),
		RawQuery: "cache=shared&mode=memory",
	})
	if err != nil {
		t.Fatalf("newDb(t): could not create in-memory database %s: %v", t.Name(), err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})
	return db
}

func Test_sqlite3Database_DHTRoutingTable(t *testing.T) {
	t.Parallel()
	db := newDb(t)
	_, filter, err := net.ParseCIDR("127.0.0.0/8")
	if err != nil {
		t.Fatal(err)
	}
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
		t.Fatalf("IsDHTRoutingEmpty() = (%v, %v), want (false, nil)", empty, err)
	}
	if nodes, err := db.DumpDHTNodes(ctx, true); err != nil || len(nodes) != 2 {
		t.Fatalf("DumpDHTNodes(IPv4) returned %d nodes and error %v, want 2 nodes and nil", len(nodes), err)
	}
	if nodes, err := db.DumpDHTNodes(ctx, false); err != nil || len(nodes) != 0 {
		t.Fatalf("DumpDHTNodes(IPv6) returned %d nodes and error %v, want 0 nodes and nil", len(nodes), err)
	}
	if nodes, err := db.GetDHTNodes(ctx); err != nil || len(nodes) != 2 {
		t.Fatalf("GetDHTNodes() returned %d nodes and error %v, want 2 nodes and nil", len(nodes), err)
	}
	if nodes, err := db.GetDHTNodes(ctx); err != nil || len(nodes) != 1 || nodes[0].Port != first.Port {
		t.Fatalf("second GetDHTNodes() = (%v, %v), want only %v", nodes, err, first)
	}
	if empty, err := db.IsDHTRoutingEmpty(ctx); err != nil || !empty {
		t.Fatalf("IsDHTRoutingEmpty() after consumption = (%v, %v), want (true, nil)", empty, err)
	}
	if err := db.AddDHTHashes(ctx, [][20]byte{{1}, {2}}); err != nil {
		t.Fatal(err)
	}
	if err := db.AddDHTHashes(ctx, [][20]byte{{3}}); err != nil {
		t.Fatal(err)
	}
	hashes, err := db.GetDHTHashes(ctx)
	if err != nil || hashes[0] != [20]byte{3} || hashes[1] != [20]byte{2} {
		t.Fatalf("GetDHTHashes() = %v, %v, want first slot replaced and second retained", hashes[:2], err)
	}
}

func Test_sqlite3Database_DHTOutbox(t *testing.T) {
	t.Parallel()
	db := newDb(t)
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
	first, found, err := db.NextDHTMessage(context.Background())
	if err != nil || !found || first.ID == "" || string(first.Payload) != "payload" {
		t.Fatalf("NextDHTMessage() = (%+v, %v, %v), want queued payload", first, found, err)
	}
	if err := db.ReleaseDHTMessage(context.Background(), first.ID); err != nil {
		t.Fatal(err)
	}
	second, found, err := db.NextDHTMessage(context.Background())
	if err != nil || !found || string(second.Payload) != "next" {
		t.Fatalf("NextDHTMessage() after release = (%+v, %v, %v), want next message", second, found, err)
	}
	if err := db.AckDHTMessage(context.Background(), second.ID); err != nil {
		t.Fatal(err)
	}
	retried, found, err := db.NextDHTMessage(context.Background())
	if err != nil || !found || retried.ID != first.ID {
		t.Fatalf("NextDHTMessage() after next ack = (%+v, %v, %v), want released message", retried, found, err)
	}
	if err := db.AckDHTMessage(context.Background(), retried.ID); err != nil {
		t.Fatal(err)
	}
	if _, found, err := db.NextDHTMessage(context.Background()); err != nil || found {
		t.Fatalf("NextDHTMessage() after ack found=%v err=%v, want no message", found, err)
	}
}

func Test_sqlite3Database_DoesTorrentExist(t *testing.T) {
	t.Parallel()
	db := newDb(t)
	tests := []struct {
		name     string
		infoHash []byte
		want     bool
		wantErr  bool
	}{
		{
			name:     "Test Empty",
			infoHash: []byte{},
			want:     false,
			wantErr:  false,
		},
		{
			name:     "Test Zeroes",
			infoHash: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			want:     false,
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := db.DoesTorrentExist(tt.infoHash)
			if (err != nil) != tt.wantErr {
				t.Errorf("sqlite3Database.DoesTorrentExist() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("sqlite3Database.DoesTorrentExist() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sqlite3Database_GetNumberOfTorrents(t *testing.T) {
	t.Parallel()
	db := newDb(t)

	got, err := db.GetNumberOfTorrents()
	if err != nil {
		t.Errorf("sqlite3Database.GetNumberOfTorrents() error = %v", err)
		return
	}
	if got != 0 {
		t.Errorf("sqlite3Database.GetNumberOfTorrents() = %v, want 0", got)
	}
}

func TestSqlite3Database_GetNumberOfQueryTorrents(t *testing.T) {
	t.Parallel()
	db := newDb(t)

	// The database is empty, so the number of torrents for any query should be 0.
	tests := []struct {
		name    string
		query   string
		epoch   int64
		want    uint64
		wantErr bool
	}{
		{
			name:    "Test Empty Query",
			query:   "",
			epoch:   0,
			want:    0,
			wantErr: false,
		},
		{
			name:    "Test Simple Query",
			query:   "test",
			epoch:   0,
			want:    0,
			wantErr: false,
		},
		{
			name:    "Test Query with Special Characters",
			query:   "test!@#$%^&*()",
			epoch:   0,
			want:    0,
			wantErr: false,
		},
		{
			name:    "Test Query with Future Epoch",
			query:   "test",
			epoch:   32503680000, // January 1, 3000
			want:    0,
			wantErr: false,
		},
		{
			name:    "Test Query with Past Epoch",
			query:   "test",
			epoch:   1000000000, // September 9, 2001
			want:    0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := db.GetNumberOfQueryTorrents(tt.query, tt.epoch)
			if (err != nil) != tt.wantErr {
				t.Errorf("sqlite3Database.GetNumberOfQueryTorrents() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("sqlite3Database.GetNumberOfQueryTorrents() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sqlite3Database_AddNewTorrent(t *testing.T) {
	t.Parallel()
	db := newDb(t)

	tests := []struct {
		name     string
		infoHash []byte
		files    []File
		wantErr  bool
	}{
		{
			name:     "Test Nil",
			infoHash: []byte{},
			files:    nil,
			wantErr:  false,
		},
		{
			name:     "Test Empty",
			infoHash: []byte{},
			files:    []File{},
			wantErr:  false,
		},
		{
			name:     "Test Zeroes",
			infoHash: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			files:    []File{{Size: 0, Path: "test"}},
			wantErr:  false,
		},
		{
			name:     "Test NonZeroes",
			infoHash: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			files:    []File{{Size: 1, Path: "test"}},
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := db.AddNewTorrent(tt.infoHash, tt.name, tt.files); (err != nil) != tt.wantErr {
				t.Errorf("sqlite3Database.AddNewTorrent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_sqlite3Database_QueryTorrents(t *testing.T) {
	t.Parallel()
	db := newDb(t)

	tests := []struct {
		name             string
		query            string
		epoch            int64
		orderBy          OrderingCriteria
		ascending        bool
		limit            uint64
		lastOrderedValue *float64
		lastID           *uint64
		want             []TorrentMetadata
		wantErr          bool
	}{
		{
			name:             "Test Relevance",
			query:            "",
			epoch:            0,
			orderBy:          ByRelevance,
			ascending:        false,
			limit:            10,
			lastOrderedValue: nil,
			lastID:           nil,
			want:             nil,
			wantErr:          true,
		},
		{
			name:             "Test DiscoveredOn",
			query:            "",
			epoch:            0,
			orderBy:          ByDiscoveredOn,
			ascending:        true,
			limit:            10,
			lastOrderedValue: nil,
			lastID:           nil,
			want:             []TorrentMetadata{},
			wantErr:          false,
		},
		{
			name:             "Test NFiles",
			query:            "",
			epoch:            0,
			orderBy:          ByNFiles,
			ascending:        false,
			limit:            10,
			lastOrderedValue: nil,
			lastID:           nil,
			want:             []TorrentMetadata{},
			wantErr:          false,
		},
		{
			name:             "Test NFiles",
			query:            "",
			epoch:            0,
			orderBy:          ByTotalSize,
			ascending:        true,
			limit:            10,
			lastOrderedValue: nil,
			lastID:           nil,
			want:             []TorrentMetadata{},
			wantErr:          false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := db.QueryTorrents(tt.query, tt.epoch, tt.orderBy, tt.ascending, tt.limit, tt.lastOrderedValue, tt.lastID)
			if (err != nil) != tt.wantErr {
				t.Errorf("sqlite3Database.QueryTorrents() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sqlite3Database.QueryTorrents() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sqlite3Database_GetTorrent(t *testing.T) {
	t.Parallel()
	db := newDb(t)

	tests := []struct {
		name     string
		infoHash []byte
		want     *TorrentMetadata
		wantErr  bool
	}{
		{
			name:     "Test Empty",
			infoHash: []byte{},
			wantErr:  false,
		},
		{
			name:     "Test Zeroes",
			infoHash: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := db.GetTorrent(tt.infoHash)
			if (err != nil) != tt.wantErr {
				t.Errorf("sqlite3Database.GetTorrent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_sqlite3Database_GetFiles(t *testing.T) {
	t.Parallel()
	db := newDb(t)

	tests := []struct {
		name     string
		infoHash []byte
		want     []File
		wantErr  bool
	}{
		{
			name:     "Test Empty",
			infoHash: []byte{},
			want:     nil,
			wantErr:  false,
		},
		{
			name:     "Test Zeroes",
			infoHash: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			want:     nil,
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := db.GetFiles(tt.infoHash)
			if (err != nil) != tt.wantErr {
				t.Errorf("sqlite3Database.GetFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sqlite3Database.GetFiles() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sqlite3Database_GetStatistics(t *testing.T) {
	t.Parallel()
	db := newDb(t)

	tests := []struct {
		name    string
		from    string
		n       uint
		want    *Statistics
		wantErr bool
	}{
		{
			name: "Test Year",
			from: "2018",
			n:    0,
			want: &Statistics{
				NDiscovered: map[string]uint64{},
				NFiles:      map[string]uint64{},
				TotalSize:   map[string]uint64{},
			},
			wantErr: false,
		},
		{
			name: "Test Month",
			from: "2018-04",
			n:    0,
			want: &Statistics{
				NDiscovered: map[string]uint64{},
				NFiles:      map[string]uint64{},
				TotalSize:   map[string]uint64{},
			},
			wantErr: false,
		},
		{
			name: "Test Week",
			from: "2018-W16",
			n:    0,
			want: &Statistics{
				NDiscovered: map[string]uint64{},
				NFiles:      map[string]uint64{},
				TotalSize:   map[string]uint64{},
			},
			wantErr: false,
		},
		{
			name: "Test Day",
			from: "2018-04-20",
			n:    0,
			want: &Statistics{
				NDiscovered: map[string]uint64{},
				NFiles:      map[string]uint64{},
				TotalSize:   map[string]uint64{},
			},
			wantErr: false,
		},
		{
			name: "Test Hour",
			from: "2018-04-20T15",
			n:    1,
			want: &Statistics{
				NDiscovered: map[string]uint64{},
				NFiles:      map[string]uint64{},
				TotalSize:   map[string]uint64{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := db.GetStatistics(tt.from, tt.n)
			if (err != nil) != tt.wantErr {
				t.Errorf("sqlite3Database.GetStatistics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sqlite3Database.GetStatistics() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sqlite3Database_Engine(t *testing.T) {
	t.Parallel()

	instance := &sqlite3Database{}
	if got := instance.Engine(); got != Sqlite3 {
		t.Errorf("zeromq.Engine() = %v, want %v", got, Sqlite3)
	}
}

func Test_sqlite3Database_Export(t *testing.T) {
	t.Parallel()
	db := newDb(t)

	infoHash1 := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	name1 := "Test Torrent 1"
	files1 := []File{{Size: 100, Path: "file1.txt"}, {Size: 200, Path: "file2.txt"}}
	err := db.AddNewTorrent(infoHash1, name1, files1)
	if err != nil {
		t.Fatalf("Failed to add torrent: %v", err)
	}

	infoHash2 := []byte{21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40}
	name2 := "Test Torrent 2"
	files2 := []File{{Size: 300, Path: "file3.txt"}}
	err = db.AddNewTorrent(infoHash2, name2, files2)
	if err != nil {
		t.Fatalf("Failed to add torrent: %v", err)
	}

	exportChan, err := db.Export()
	if err != nil {
		t.Fatalf("Export returned an error: %v", err)
	}

	var summaries []SimpleTorrentSummary
	for summary := range exportChan {
		summaries = append(summaries, summary)
	}

	if len(summaries) != 2 {
		t.Fatalf("Expected 2 summaries, got %d", len(summaries))
	}

	expectedHash1 := "0102030405060708090a0b0c0d0e0f1011121314"
	if summaries[0].InfoHash != expectedHash1 {
		t.Errorf("Summary 1: Expected InfoHash %s, got %s", expectedHash1, summaries[0].InfoHash)
	}
	if summaries[0].Name != name1 {
		t.Errorf("Summary 1: Expected Name %s, got %s", name1, summaries[0].Name)
	}
	if !reflect.DeepEqual(summaries[0].Files, files1) {
		t.Errorf("Summary 1: Expected Files %v, got %v", files1, summaries[0].Files)
	}

	expectedHash2 := "15161718191a1b1c1d1e1f202122232425262728"
	if summaries[1].InfoHash != expectedHash2 {
		t.Errorf("Summary 2: Expected InfoHash %s, got %s", expectedHash2, summaries[1].InfoHash)
	}
	if summaries[1].Name != name2 {
		t.Errorf("Summary 2: Expected Name %s, got %s", name2, summaries[1].Name)
	}
	if !reflect.DeepEqual(summaries[1].Files, files2) {
		t.Errorf("Summary 2: Expected Files %v, got %v", files2, summaries[1].Files)
	}
}
