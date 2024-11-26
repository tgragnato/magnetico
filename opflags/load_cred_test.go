package opflags

import (
	"os"
	"reflect"
	"testing"
)

func TestLoadCred(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		credContent string
		expectedErr bool
		expectedMap map[string][]byte
	}{
		{
			name:        "Invalid credentials format",
			credContent: "user1:invalidhash\n",
			expectedErr: true,
			expectedMap: nil,
		},
		{
			name:        "Empty credentials",
			credContent: "",
			expectedErr: false,
			expectedMap: map[string][]byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file with the test credentials
			tmpfile, err := os.CreateTemp("", "testcred")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())

			if _, err := tmpfile.Write([]byte(tt.credContent)); err != nil {
				t.Fatal(err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatal(err)
			}

			// Initialize OpFlags with the temporary file path
			o := &OpFlags{
				Cred:        tmpfile.Name(),
				Credentials: make(map[string][]byte),
			}

			// Call LoadCred and check the result
			err = o.LoadCred()
			if (err != nil) != tt.expectedErr {
				t.Errorf("LoadCred() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}

			if !tt.expectedErr && !reflect.DeepEqual(o.Credentials, tt.expectedMap) {
				t.Errorf("LoadCred() credentials = %v, expectedMap %v", o.Credentials, tt.expectedMap)
			}
		})
	}
}
