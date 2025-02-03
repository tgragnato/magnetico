package web

import (
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sync"
	"testing"

	"tgragnato.it/magnetico/persistence"
)

func TestInfohashMiddleware(t *testing.T) {
	t.Parallel()

	inputV1 := "1234567890123456789012345678901234567890"
	v1, err := hex.DecodeString(inputV1)
	if err != nil {
		t.Fatalf("error decoding infohash: %v", err)
	}

	inputV2 := "123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0"
	v2, err := hex.DecodeString(inputV2)
	if err != nil {
		t.Fatalf("error decoding infohash v2: %v", err)
	}

	tests := []struct {
		name             string
		urlInfohash      string
		expectedStatus   int
		expectedInfohash []byte
	}{
		{
			name:             "Valid Infohash v1",
			urlInfohash:      inputV1,
			expectedStatus:   http.StatusOK,
			expectedInfohash: v1,
		},
		{
			name:             "Valid Infohash v2",
			urlInfohash:      inputV2,
			expectedStatus:   http.StatusOK,
			expectedInfohash: v2,
		},
		{
			name:             "Invalid Infohash",
			urlInfohash:      "invalidinfohash",
			expectedStatus:   http.StatusBadRequest,
			expectedInfohash: []byte(""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{}
			req.SetPathValue("infohash", tt.urlInfohash)

			rr := httptest.NewRecorder()
			handler := infohashMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				infohash := r.Context().Value(InfohashKey).([]byte)
				if !reflect.DeepEqual(infohash, tt.expectedInfohash) {
					t.Errorf("expected infohash %v, got %v", tt.expectedInfohash, infohash)
				}
				w.WriteHeader(http.StatusOK)
			}))

			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}
		})
	}
}

func TestRobotsHandler(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
	rr := httptest.NewRecorder()

	robotsHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if ct := rr.Header().Get(ContentType); ct != "text/plain" {
		t.Errorf("expected Content-Type text/plain, got %q", ct)
	}
	if body := rr.Body.String(); body != "User-agent: *\nDisallow: /\n" {
		t.Error("got unexpected body")
	}
}

var initMux sync.Mutex

func initDb() {
	initMux.Lock()
	defer initMux.Unlock()

	if database != nil {
		return
	}

	dbUrl := url.URL{
		Scheme:   "sqlite3",
		Path:     "/web.db",
		RawQuery: "cache=shared&mode=memory",
	}
	database, _ = persistence.MakeDatabase(dbUrl.String())
}
