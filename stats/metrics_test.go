package stats

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetInstance(t *testing.T) {
	t.Parallel()

	instance1 := GetInstance()
	instance2 := GetInstance()
	if instance1 != instance2 {
		t.Error("Expected GetInstance to return the same instance")
	}

	if instance1.extensions == nil {
		t.Error("Expected GetInstance to initialize the extensions map")
	}
}

func TestMakePrometheusHandler(t *testing.T) {
	t.Parallel()

	handler := MakePrometheusHandler()
	if handler == nil {
		t.Error("Expected MakePrometheusHandler to return a valid http.HandlerFunc")
	}

	req, err := http.NewRequest("GET", "/metrics", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	wanted := "text/plain; version=0.0.4; charset=utf-8; escaping=underscores"
	if contentType := rr.Header().Get("Content-Type"); contentType != wanted {
		t.Errorf("Handler returned wrong content type: got %v want %v", contentType, wanted)
	}

	if body := rr.Body.String(); body == "" {
		t.Error("Expected handler to return a non-empty body")
	}
}
