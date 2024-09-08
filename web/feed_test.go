package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFeedHandler(t *testing.T) {
	t.Parallel()

	initDb()

	tests := []struct {
		name           string
		query          string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "No query",
			query:          "",
			expectedStatus: http.StatusOK,
			expectedBody: "<?xml version=\"1.0\" encoding=\"utf-8\" standalone=\"yes\"?>\n" +
				"<rss version=\"2.0\"><Channel><item><title>Most recent torrents - magnetico</title></item></Channel></rss>",
		},
		{
			name:           "Single query",
			query:          "test",
			expectedStatus: http.StatusOK,
			expectedBody: "<?xml version=\"1.0\" encoding=\"utf-8\" standalone=\"yes\"?>\n" +
				"<rss version=\"2.0\"><Channel><item><title>test - magnetico</title></item></Channel></rss>",
		},
		{
			name:           "Multiple queries",
			query:          "test1&query=test2",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "query supplied multiple times!\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/feed?query="+tt.query, nil)
			rr := httptest.NewRecorder()

			feedHandler(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK {
				if rr.Body.String() == "" {
					t.Errorf("handler returned empty body")
				}
			}

			if tt.expectedBody != rr.Body.String() {
				t.Errorf(
					"handler returned unexpected body: got %v want %v",
					rr.Body.String(),
					tt.expectedBody,
				)
			}
		})
	}
}
