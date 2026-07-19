package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tgragnato.it/magnetico/v2/persistence"
)

func TestStatistics(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	if err := statistics(persistence.NewStatistics()).Render(rec); err != nil {
		t.Errorf("statistics render: %v", err)
	}
}

func TestStatisticsWithData(t *testing.T) {
	t.Parallel()

	stats := persistence.NewStatistics()
	stats.NDiscovered["2024-01"] = 42
	stats.NFiles["2024-01"] = 100
	stats.TotalSize["2024-01"] = 1024 * 1024 * 1024

	rec := httptest.NewRecorder()
	if err := statistics(stats).Render(rec); err != nil {
		t.Errorf("statistics render: %v", err)
	}
	html := rec.Body.String()
	if !strings.Contains(html, "Torrents Discovered") {
		t.Errorf("chart title not found in HTML")
	}
}

func TestStatisticsHandler(t *testing.T) {
	t.Parallel()

	initDb()

	req, err := http.NewRequest("GET", "/statistics", nil)
	if err != nil {
		t.Fatalf("could not create request: %v", err)
	}

	rec := httptest.NewRecorder()
	handler := http.HandlerFunc(statisticsHandler)
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

func TestStatisticsPartialHandler(t *testing.T) {
	t.Parallel()

	initDb()

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
	}{
		{"default params", "", http.StatusOK},
		{"hours", "n=24&unit=hours", http.StatusOK},
		{"days", "n=7&unit=days", http.StatusOK},
		{"weeks", "n=4&unit=weeks", http.StatusOK},
		{"months", "n=3&unit=months", http.StatusOK},
		{"years", "n=1&unit=years", http.StatusOK},
		{"invalid unit", "n=24&unit=invalid", http.StatusBadRequest},
		{"invalid n", "n=abc&unit=hours", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/statistics/partial?"+tt.queryParams, nil)
			if err != nil {
				t.Fatalf("could not create request: %v", err)
			}

			rec := httptest.NewRecorder()
			handler := http.HandlerFunc(statisticsPartialHandler)
			handler.ServeHTTP(rec, req)

			res := rec.Result()
			defer res.Body.Close()

			if res.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %v; got %v", tt.expectedStatus, res.StatusCode)
			}
		})
	}
}

func TestAPIStatistics(t *testing.T) {
	t.Parallel()

	initDb()

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "valid request",
			queryParams:    "?from=2023-01-01&n=10",
			expectedStatus: http.StatusOK,
			expectedBody:   "{\"nDiscovered\":{},\"nFiles\":{},\"totalSize\":{}}\n",
		},
		{
			name:           "missing n",
			queryParams:    "?from=2023-01-01",
			expectedStatus: http.StatusOK,
			expectedBody:   "{\"nDiscovered\":{},\"nFiles\":{},\"totalSize\":{}}\n",
		},
		{
			name:           "invalid n",
			queryParams:    "?from=2023-01-01&n=invalid",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Couldn't parse n: strconv.ParseInt: parsing \"invalid\": invalid syntax\n",
		},
		{
			name:           "negative n",
			queryParams:    "?from=2023-01-01&n=-5",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "n must be a positive number\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/statistics"+tt.queryParams, nil)
			if err != nil {
				t.Fatalf("could not create request: %v", err)
			}

			rec := httptest.NewRecorder()
			handler := http.HandlerFunc(apiStatistics)
			handler.ServeHTTP(rec, req)

			res := rec.Result()
			defer res.Body.Close()

			if res.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %v; got %v", tt.expectedStatus, res.StatusCode)
			}

			if tt.expectedStatus == http.StatusOK {
				contentType := res.Header.Get("Content-Type")
				if contentType != "application/json; charset=utf-8" {
					t.Errorf("expected Content-Type application/json; got %v", contentType)
				}
			}

			if rec.Body.String() != tt.expectedBody {
				t.Errorf("expected %s; got %s", tt.expectedBody, rec.Body.String())
			}
		})
	}
}
