package web

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStatistics(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	if err := statistics().Render(&buffer); err != nil {
		t.Errorf("statistics render: %v", err)
	}
}

func TestStatisticsHandler(t *testing.T) {
	t.Parallel()

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
