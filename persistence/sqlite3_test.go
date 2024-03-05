package persistence

import (
	"net/url"
	"reflect"
	"testing"
	"text/template"

	_ "modernc.org/sqlite"
)

var db, _ = makeSqlite3Database(&url.URL{Scheme: "sqlite", Path: ":memory:?cache=shared"})

func TestSqlite3Database_ExecuteTemplate(t *testing.T) {
	t.Parallel()

	text := "Hello, {{.Name}}!"
	data := struct {
		Name string
	}{
		Name: "World",
	}

	expected := "Hello, World!"

	result := executeTemplate(text, data, template.FuncMap{})
	if result != expected {
		t.Errorf("Expected result to be %q, but got %q", expected, result)
	}
}

func Test_makeSqlite3Database(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		url_    *url.URL
		want    bool
		wantErr bool
	}{
		{
			name: "Test sqlite3",
			url_: &url.URL{
				Scheme: "sqlite3",
				Path:   ":memory:",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "Test sqlite",
			url_: &url.URL{
				Scheme: "sqlite",
				Path:   ":memory:?cache=shared",
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := makeSqlite3Database(tt.url_)
			if (err != nil) != tt.wantErr {
				t.Errorf("makeSqlite3Database() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (got != nil) != tt.want {
				t.Error("makeSqlite3Database() == nil, want != nil")
			}
		})
	}
}

func Test_sqlite3Database_DoesTorrentExist(t *testing.T) {
	t.Parallel()

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
			name:     "Test InfoHash",
			infoHash: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0},
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

	t.Run("Test GetNumberOfTorrents on empty database", func(t *testing.T) {
		got, err := db.GetNumberOfTorrents()
		if err != nil {
			t.Errorf("sqlite3Database.GetNumberOfTorrents() error = %v", err)
			return
		}
		if got != 0 {
			t.Errorf("sqlite3Database.GetNumberOfTorrents() = %v, want 0", got)
		}
	})
}

func Test_sqlite3Database_QueryTorrents(t *testing.T) {
	t.Parallel()

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
			name:             "Test Empty ByNFiles",
			query:            "",
			epoch:            0,
			orderBy:          ByNFiles,
			ascending:        true,
			limit:            10,
			lastOrderedValue: nil,
			lastID:           nil,
			want:             []TorrentMetadata{},
			wantErr:          false,
		},
		{
			name:             "Test Empty ByDiscoveredOn",
			query:            "",
			epoch:            0,
			orderBy:          ByDiscoveredOn,
			ascending:        false,
			limit:            10,
			lastOrderedValue: nil,
			lastID:           nil,
			want:             []TorrentMetadata{},
			wantErr:          false,
		},
		{
			name:             "Test Empty ByTotalSize",
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
		{
			name:             "Test Empty ByRelevance",
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

func Test_sqlite3Database_GetStatistics(t *testing.T) {
	t.Parallel()

	for _, tt := range validDates {
		t.Run(tt.date, func(t *testing.T) {
			_, err := db.GetStatistics(tt.date, 0)
			if err != nil {
				t.Errorf("sqlite3Database.GetStatistics() error = %v", err)
				return
			}
		})
	}
}

func Test_sqlite3Database_GetTorrent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		infoHash []byte
		want     *TorrentMetadata
		wantErr  bool
	}{
		{
			name:     "Test Bad InfoHash",
			infoHash: []byte{},
			want:     nil,
			wantErr:  false,
		},
		{
			name:     "Test Good InfoHash",
			infoHash: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0},
			want:     nil,
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := db.GetTorrent(tt.infoHash)
			if (err != nil) != tt.wantErr {
				t.Errorf("sqlite3Database.GetTorrent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sqlite3Database.GetTorrent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sqlite3Database_GetFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		infoHash []byte
		want     []File
		wantErr  bool
	}{
		{
			name:     "Test Bad InfoHash",
			infoHash: []byte{},
			want:     nil,
			wantErr:  false,
		},
		{
			name:     "Test Good InfoHash",
			infoHash: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0},
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

func Test_sqlite3Database_AddNewTorrent(t *testing.T) {
	tests := []struct {
		name     string
		infoHash []byte
		files    []File
		wantErr  bool
	}{
		{
			name:     "Test Empty",
			infoHash: []byte{},
			files:    []File{},
			wantErr:  false,
		},
		{
			name:     "Test File",
			infoHash: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0},
			files:    []File{{Size: 100, Path: "test"}},
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
