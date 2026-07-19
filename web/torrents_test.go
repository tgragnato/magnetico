package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"tgragnato.it/magnetico/v2/persistence"
)

func TestTorrents(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	if err := torrents("", time.Now().Unix(), "DISCOVERED_ON", false, nil).Render(rec); err != nil {
		t.Errorf("torrents render: %v", err)
	}
}

func TestTorrentsWithQuery(t *testing.T) {
	t.Parallel()

	results := []persistence.TorrentMetadata{
		{ID: 1, InfoHash: make([]byte, 20), Name: "Test Torrent", Size: 1024, DiscoveredOn: 1710000000, NFiles: 1},
	}
	rec := httptest.NewRecorder()
	if err := torrents("hello", time.Now().Unix(), "RELEVANCE", true, results).Render(rec); err != nil {
		t.Errorf("torrents render: %v", err)
	}
	html := rec.Body.String()
	if !strings.Contains(html, "hello - magnetico") {
		t.Errorf("title not found in HTML")
	}
	if !strings.Contains(html, "Test Torrent") {
		t.Errorf("torrent name not found in HTML")
	}
}

func TestTorrentsHandler(t *testing.T) {
	t.Parallel()

	initDb()

	req, err := http.NewRequest("GET", "/torrents", nil)
	if err != nil {
		t.Fatalf("could not create request: %v", err)
	}

	rec := httptest.NewRecorder()
	handler := http.HandlerFunc(torrentsHandler)
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

func TestTorrentsResultsHandler(t *testing.T) {
	t.Parallel()

	initDb()

	req, err := http.NewRequest("GET", "/torrents/results", nil)
	if err != nil {
		t.Fatalf("could not create request: %v", err)
	}

	rec := httptest.NewRecorder()
	handler := http.HandlerFunc(torrentsResultsHandler)
	handler.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status OK; got %v", res.Status)
	}
}

func TestApiTorrents(t *testing.T) {
	t.Parallel()

	initDb()

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "valid request without optional params",
			queryParams:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid request with only lastOrderedValue",
			queryParams:    "lastOrderedValue=123.45",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "`lastOrderedValue`, `lastID` must be supplied altogether, if supplied.",
		},
		{
			name:           "invalid request with only lastID",
			queryParams:    "lastID=123",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "`lastOrderedValue`, `lastID` must be supplied altogether, if supplied.",
		},
		{
			name:           "valid request with lastOrderedValue and lastID",
			queryParams:    "lastOrderedValue=123.45&lastID=123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid request with non-numeric epoch",
			queryParams:    "epoch=abc",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "error while parsing the URL: strconv.ParseInt: parsing \"abc\": invalid syntax",
		},
		{
			name:           "invalid request with non-boolean ascending",
			queryParams:    "ascending=notabool",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "error while parsing the URL: strconv.ParseBool: parsing \"notabool\": invalid syntax",
		},
		{
			name:           "invalid request with non-numeric limit",
			queryParams:    "limit=notanumber",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "error while parsing the URL: strconv.ParseUint: parsing \"notanumber\": invalid syntax",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/torrents?"+tt.queryParams, nil)
			if err != nil {
				t.Fatalf("could not create request: %v", err)
			}

			rec := httptest.NewRecorder()
			handler := http.HandlerFunc(apiTorrents)
			handler.ServeHTTP(rec, req)

			res := rec.Result()
			defer res.Body.Close()

			if res.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %v; got %v", tt.expectedStatus, res.StatusCode)
			}

			if tt.expectedError != "" {
				if !strings.Contains(rec.Body.String(), tt.expectedError) {
					t.Errorf("expected error %q; got %q", tt.expectedError, rec.Body.String())
				}
			}
		})
	}
}

func TestParseOrderBy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input          string
		expectedOutput persistence.OrderingCriteria
		expectedError  bool
	}{
		{"RELEVANCE", persistence.ByRelevance, false},
		{"TOTAL_SIZE", persistence.ByTotalSize, false},
		{"DISCOVERED_ON", persistence.ByDiscoveredOn, false},
		{"N_FILES", persistence.ByNFiles, false},
		{"UPDATED_ON", persistence.ByUpdatedOn, false},
		{"N_SEEDERS", persistence.ByNSeeders, false},
		{"N_LEECHERS", persistence.ByNLeechers, false},
		{"UNKNOWN", persistence.ByDiscoveredOn, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			output, err := parseOrderBy(tt.input)
			if (err != nil) != tt.expectedError {
				t.Errorf("expected error: %v, got: %v", tt.expectedError, err)
			}
			if output != tt.expectedOutput {
				t.Errorf("expected output: %v, got: %v", tt.expectedOutput, output)
			}
		})
	}
}

func TestApiTorrentsTotal(t *testing.T) {
	t.Parallel()

	initDb()

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "missing required epoch parameter",
			queryParams:    "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "lack required parameters while parsing the URL: `epoch`",
		},
		{
			name:           "invalid epoch parameter",
			queryParams:    "epoch=abc",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "error while parsing the URL: strconv.ParseInt: parsing \"abc\": invalid syntax",
		},
		{
			name:           "valid request with epoch",
			queryParams:    "epoch=1234567890&query=testQuery",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid request with only lastOrderedValue",
			queryParams:    "epoch=1234567890&lastOrderedValue=123.45",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "`lastOrderedValue`, `lastID` must be supplied altogether, if supplied.",
		},
		{
			name:           "invalid request with only lastID",
			queryParams:    "epoch=1234567890&lastID=123",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "`lastOrderedValue`, `lastID` must be supplied altogether, if supplied.",
		},
		{
			name:           "valid request with both lastOrderedValue and lastID",
			queryParams:    "epoch=1234567890&lastOrderedValue=123.45&lastID=123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid request with newLogic=true",
			queryParams:    "epoch=1234567890&newLogic=true&queryType=byKeyword",
			expectedStatus: http.StatusOK,
			expectedError:  `{"data":0,"queryType":"byKeyword"}`,
		},
		{
			name:           "invalid queryType",
			queryParams:    "epoch=1234567890&newLogic=true&queryType=invalidType",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "error while parsing the URL: unknown queryType string: invalidType",
		},
		{
			name:           "invalid newLogic parameter",
			queryParams:    "epoch=1234567890&newLogic=bool",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "error while parsing the URL: strconv.ParseBool: parsing \"bool\": invalid syntax",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/torrentstotal?"+tt.queryParams, nil)
			if err != nil {
				t.Fatalf("could not create request: %v", err)
			}

			rec := httptest.NewRecorder()
			handler := http.HandlerFunc(apiTorrentsTotal)
			handler.ServeHTTP(rec, req)

			res := rec.Result()
			defer res.Body.Close()

			if res.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %v; got %v", tt.expectedStatus, res.StatusCode)
			}

			if tt.expectedError != "" {
				if !strings.Contains(rec.Body.String(), tt.expectedError) {
					t.Errorf("expected error %q; got %q", tt.expectedError, rec.Body.String())
				}
			}
		})
	}
}

func TestParseQueryCountType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          string
		expectedOutput CountQueryTorrentsType
		expectedError  string
	}{
		{
			name:           "Valid byKeyword",
			input:          "byKeyword",
			expectedOutput: CountQueryTorrentsByKeyword,
			expectedError:  "",
		},
		{
			name:           "Valid byAll",
			input:          "byAll",
			expectedOutput: CountQueryTorrentsByAll,
			expectedError:  "",
		},
		{
			name:           "Invalid queryType",
			input:          "invalidType",
			expectedOutput: CountQueryTorrentsByKeyword,
			expectedError:  "unknown queryType string: invalidType",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := parseQueryCountType(tt.input)

			if err != nil && err.Error() != tt.expectedError {
				t.Errorf("expected error %v, got %v", tt.expectedError, err.Error())
			} else if err == nil && tt.expectedError != "" {
				t.Errorf("expected error %v, got nil", tt.expectedError)
			}

			if output != tt.expectedOutput {
				t.Errorf("expected output %v, got %v", tt.expectedOutput, output)
			}
		})
	}
}
