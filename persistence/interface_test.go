package persistence

import (
	"encoding/json"
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
