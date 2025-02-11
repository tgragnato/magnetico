package persistence

import (
	"encoding/hex"
	"encoding/json"
	"net/url"
	"os"
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

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			if i > start {
				lines = append(lines, data[start:i])
			}
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}

func TestMakeExport_WithContent_JSON(t *testing.T) {
	t.Parallel()

	exportFile, err := os.CreateTemp("", "test_content_export.json")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	exportPath := exportFile.Name()
	os.Remove(exportPath)
	exportFile.Close()

	infoHash := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	data := []SimpleTorrentSummary{
		{
			InfoHash: hex.EncodeToString(infoHash),
			Name:     "content_torrent",
			Files:    []File{{Path: "fileA.txt", Size: 150}},
		},
	}

	db := newDb(t)
	for _, st := range data {
		if err := db.AddNewTorrent(infoHash, st.Name, st.Files); err != nil {
			t.Fatalf("Failed to add torrent to database: %v", err)
		}
	}

	err = MakeExport(db, exportPath, make(chan os.Signal, 1))
	if err != nil {
		t.Fatalf("MakeExport() error = %v", err)
	}

	fileContent, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("Failed to read export file: %v", err)
	}

	if len(fileContent) == 0 {
		t.Errorf("Expected non-empty export file for JSON, got empty")
	}

	lines := splitLines(fileContent)
	if len(lines) != len(data) {
		t.Errorf("Expected %d JSON lines, got %d", len(data), len(lines))
	}
	for _, line := range lines {
		var sts SimpleTorrentSummary
		if err = json.Unmarshal(line, &sts); err != nil {
			t.Errorf("Failed to unmarshal JSON line: %v", err)
		}
	}
}

func TestMakeImport_WithContent_JSON(t *testing.T) {
	t.Parallel()

	importFile, err := os.CreateTemp("", "test_content_import.json")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	importPath := importFile.Name()
	defer os.Remove(importPath)
	defer importFile.Close()

	infoHash := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	data := []SimpleTorrentSummary{
		{
			InfoHash: hex.EncodeToString(infoHash),
			Name:     "content_torrent",
			Files:    []File{{Path: "fileA.txt", Size: 150}},
		},
	}

	for _, st := range data {
		jsonData, err := json.Marshal(st)
		if err != nil {
			t.Fatalf("Failed to marshal test data: %v", err)
		}
		jsonData = append(jsonData, '\n')
		if _, err := importFile.Write(jsonData); err != nil {
			t.Fatalf("Failed to write to import file: %v", err)
		}
	}

	db := newDb(t)
	err = MakeImport(db, importPath, make(chan os.Signal, 1))
	if err != nil {
		t.Fatalf("MakeImport() error = %v", err)
	}

	exist, err := db.DoesTorrentExist(infoHash)
	if err != nil {
		t.Fatalf("Failed to check if torrent exists: %v", err)
	}
	if !exist {
		t.Error("Expected imported torrent to exist in database")
	}

	files, err := db.GetFiles(infoHash)
	if err != nil {
		t.Fatalf("Failed to get files: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(files))
	}
	if files[0].Path != "fileA.txt" || files[0].Size != 150 {
		t.Errorf("Unexpected file data: %+v", files[0])
	}
}
