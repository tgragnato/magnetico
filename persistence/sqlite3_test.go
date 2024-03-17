package persistence

import (
	"net/url"
	"reflect"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func newDb(t *testing.T) Database {
	db, e := makeSqlite3Database(&url.URL{
		Scheme:   "sqlite3",
		Path:     t.Name(),
		RawQuery: "cache=shared&mode=memory",
	})
	if e != nil {
		t.Errorf("newDb(t): Could not create in-memory database %s. error = %v", t.Name(), e)
	}
	return db
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
		limit            uint
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
