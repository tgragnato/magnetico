package persistence

import (
	"encoding/json"
	"net/url"
	"testing"
)

func TestTorrentMetadata_MarshalJSON(t *testing.T) {
	t.Parallel()

	tm := &TorrentMetadata{
		InfoHash: []byte{1, 2, 3, 4, 5, 6},
	}

	expectedJSON := `{"infoHash":"010203040506","id":0,"name":"","size":0,"discoveredOn":0,"nFiles":0,"relevance":0}`

	jsonData, err := tm.MarshalJSON()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}

	if string(jsonData) != expectedJSON {
		t.Errorf("Unexpected JSON string. Expected: %s, Got: %s", expectedJSON, string(jsonData))
	}
}

func TestNewStatistics(t *testing.T) {
	t.Parallel()

	s := NewStatistics()

	if s.NDiscovered == nil {
		t.Error("NDiscovered map is not initialized")
	}

	if s.NFiles == nil {
		t.Error("NFiles map is not initialized")
	}

	if s.TotalSize == nil {
		t.Error("TotalSize map is not initialized")
	}
}

func TestMakeDatabase(t *testing.T) {
	t.Parallel()

	sqlite3URL := url.URL{
		Scheme:   "sqlite3",
		Path:     "/makedatabasesqlite3.db",
		RawQuery: "cache=shared&mode=memory",
	}
	sqliteURL := url.URL{
		Scheme:   "sqlite",
		Path:     "/makedatabasesqlite.db",
		RawQuery: "cache=shared&mode=memory",
	}
	bitmagnetURL := url.URL{
		Scheme:  "bitmagnet",
		Path:    "tgragnato.it/import",
		RawPath: "debug=true&source=testsource",
	}
	bitmagnetsURL := url.URL{
		Scheme:  "bitmagnets",
		Path:    "tgragnato.it/import",
		RawPath: "debug=true&source=testsource",
	}

	tests := []struct {
		rawURL       string
		expectError  bool
		expectedType databaseEngine
	}{
		{sqlite3URL.String(), false, Sqlite3},
		{sqliteURL.String(), false, Sqlite3},
		// Can't test Postgres without a running server
		// Can't test CockroachDB without a running server
		// Can't test ZeroMQ without attaching a zsock
		// Can't test RabbitMQ without attaching a rconn
		{bitmagnetURL.String(), false, Bitmagnet},
		{bitmagnetsURL.String(), false, Bitmagnet},
		{"invalidscheme://localhost", true, 0},
	}

	for _, tt := range tests {
		db, err := MakeDatabase(tt.rawURL)
		if tt.expectError {
			if err == nil {
				t.Errorf("Expected error for URL %s, but got none", tt.rawURL)
			}
			continue
		}

		if err != nil {
			t.Errorf("Unexpected error for URL %s: %v", tt.rawURL, err)
			continue
		}

		if db.Engine() != tt.expectedType {
			t.Errorf("Unexpected database engine for URL %s. Expected: %v, Got: %v", tt.rawURL, tt.expectedType, db.Engine())
		}
	}
}
