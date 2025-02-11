package stats

import (
	"os"
	"testing"
)

func TestInitPyroscope(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		serverAddr    string
		wantErr       bool
		expectedError string
	}{
		{
			name:       "Valid server address",
			serverAddr: "http://localhost:4040",
			wantErr:    false,
		},
		{
			name:          "Empty server address",
			serverAddr:    "",
			wantErr:       true,
			expectedError: "pyroscope server address is required",
		},
	}

	origHostname := os.Getenv("HOSTNAME")
	os.Setenv("HOSTNAME", "test-host")
	defer os.Setenv("HOSTNAME", origHostname)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profiler, err := InitPyroscope(tt.serverAddr)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				if err != nil && err.Error() != tt.expectedError {
					t.Errorf("expected error %v, got %v", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if profiler == nil {
					t.Error("expected non-nil profiler")
				}
			}

			if profiler != nil {
				if err := profiler.Stop(); err != nil {
					t.Errorf("error stopping profiler: %v", err)
				}
			}
		})
	}
}
