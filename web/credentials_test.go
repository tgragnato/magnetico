package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestBasicAuth(t *testing.T) {
	password := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	validCredentials := map[string][]byte{"testuser": hashedPassword}
	defer UpdateCredentials(nil)

	tests := []struct {
		name           string
		credentials    map[string][]byte
		username       string
		password       string
		expectedStatus int
	}{
		{
			name:           "Valid credentials",
			credentials:    validCredentials,
			username:       "testuser",
			password:       "password123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid username",
			credentials:    validCredentials,
			username:       "invaliduser",
			password:       "password123",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid password",
			credentials:    validCredentials,
			username:       "testuser",
			password:       "wrongpassword",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "No credentials",
			credentials:    validCredentials,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Empty credentials map still requires authentication",
			credentials:    map[string][]byte{},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Nil credentials disables authentication",
			credentials:    nil,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			UpdateCredentials(tt.credentials)
			req, err := http.NewRequest("GET", "/", nil)
			if err != nil {
				t.Fatal(err)
			}

			if tt.username != "" && tt.password != "" {
				req.SetBasicAuth(tt.username, tt.password)
			}

			rr := httptest.NewRecorder()
			handler := basicAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}
		})
	}
}

func TestBasicAuthUsesUpdatedCredentials(t *testing.T) {
	oldHash, err := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte("new-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	defer UpdateCredentials(nil)

	handler := basicAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	requestStatus := func(password string) int {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.SetBasicAuth("testuser", password)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		return rr.Code
	}

	UpdateCredentials(map[string][]byte{"testuser": oldHash})
	if got := requestStatus("old-password"); got != http.StatusOK {
		t.Fatalf("old credential status = %d, want %d", got, http.StatusOK)
	}

	UpdateCredentials(map[string][]byte{"testuser": newHash})
	if got := requestStatus("old-password"); got != http.StatusUnauthorized {
		t.Errorf("revoked credential status = %d, want %d", got, http.StatusUnauthorized)
	}
	if got := requestStatus("new-password"); got != http.StatusOK {
		t.Errorf("new credential status = %d, want %d", got, http.StatusOK)
	}
}
