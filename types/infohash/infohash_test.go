package infohash

import (
	"bytes"
	"fmt"
	"testing"
)

func TestT_Format(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		hash     T
		expected string
	}{
		{
			name:     "Empty Hash",
			hash:     T{},
			expected: "0000000000000000000000000000000000000000",
		},
		{
			name:     "Non-Empty Hash",
			hash:     T{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
			expected: "0102030405060708090a0b0c0d0e0f1011121314",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := fmt.Sprintf("%v", tc.hash)
			if result != tc.expected {
				t.Errorf("Expected format: %s, but got: %s", tc.expected, result)
			}
		})
	}
}

func TestT_String_FromHexString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tr   T
	}{
		{
			name: "Empty Hash",
			tr:   T{},
		},
		{
			name: "Non-Empty Hash",
			tr:   T{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := T{}
			if err := got.FromHexString(tt.tr.String()); err != nil {
				t.Errorf("T.FromHexString() failed with error: %s", err.Error())
			}
			if got != tt.tr {
				t.Errorf("Expected hash: %s, but got: %s", tt.tr, got)
			}
		})
	}
}

func TestT_IsZero(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tr   *T
		want bool
	}{
		{
			name: "Empty Hash",
			tr:   &T{},
			want: true,
		},
		{
			name: "Non-Empty Hash",
			tr:   &T{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tr.IsZero(); got != tt.want {
				t.Errorf("T.IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestT_UnmarshalText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		tr      *T
		b       []byte
		wantErr bool
	}{
		{
			name:    "Empty Hash",
			tr:      &T{},
			b:       []byte(""),
			wantErr: true,
		},
		{
			name:    "Non-Empty Hash",
			tr:      &T{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
			b:       []byte("0102030405060708090a0b0c0d0e0f1011121314"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.tr.UnmarshalText(tt.b); (err != nil) != tt.wantErr {
				t.Errorf("T.UnmarshalText() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestT_MarshalText(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		hash     T
		expected []byte
	}{
		{
			name:     "Empty Hash",
			hash:     T{},
			expected: []byte("0000000000000000000000000000000000000000"),
		},
		{
			name:     "Non-Empty Hash",
			hash:     T{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
			expected: []byte("0102030405060708090a0b0c0d0e0f1011121314"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.hash.MarshalText()
			if err != nil {
				t.Errorf("T.MarshalText() returned an unexpected error: %s", err.Error())
			}
			if !bytes.Equal(result, tc.expected) {
				t.Errorf("Expected marshaled text: %s, but got: %s", tc.expected, result)
			}
		})
	}
}

func TestFromHexString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected T
	}{
		{
			name:     "Empty String",
			input:    "",
			expected: T{},
		},
		{
			name:     "Valid Hex String",
			input:    "0102030405060708090a0b0c0d0e0f1011121314",
			expected: T{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
		},
		{
			name:     "Invalid Hex String",
			input:    "golang",
			expected: T{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := FromHexString(tc.input)
			if result != tc.expected {
				t.Errorf("Expected hash: %v, but got: %v", tc.expected, result)
			}
		})
	}
}

func TestHashBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []byte
		expected T
	}{
		{
			name:     "Empty Input",
			input:    []byte{},
			expected: T{0xda, 0x39, 0xa3, 0xee, 0x5e, 0x6b, 0x4b, 0x0d, 0x32, 0x55, 0xbf, 0xef, 0x95, 0x60, 0x18, 0x90, 0xaf, 0xd8, 0x07, 0x09},
		},
		{
			name:     "Non-Empty Input",
			input:    []byte("test"),
			expected: T{0xa9, 0x4a, 0x8f, 0xe5, 0xcc, 0xb1, 0x9b, 0xa6, 0x1c, 0x4c, 0x08, 0x73, 0xd3, 0x91, 0xe9, 0x87, 0x98, 0x2f, 0xbb, 0xd3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HashBytes(tt.input)
			if result != tt.expected {
				t.Errorf("Expected hash: %v, but got: %v", tt.expected, result)
			}
		})
	}
}

func TestHashBytesV2(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []byte
		expected T
	}{
		{
			name:     "Empty Input",
			input:    []byte{},
			expected: T{0xe3, 0xb0, 0xc4, 0x42, 0x98, 0xfc, 0x1c, 0x14, 0x9a, 0xfb, 0xf4, 0xc8, 0x99, 0x6f, 0xb9, 0x24, 0x27, 0xae, 0x41, 0xe4},
		},
		{
			name:     "Non-Empty Input",
			input:    []byte("test"),
			expected: T{0x9f, 0x86, 0xd0, 0x81, 0x88, 0x4c, 0x7d, 0x65, 0x9a, 0x2f, 0xea, 0xa0, 0xc5, 0x5a, 0xd0, 0x15, 0xa3, 0xbf, 0x4f, 0x1b},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HashBytesV2(tt.input)
			if result != tt.expected {
				t.Errorf("Expected hash: %v, but got: %v", tt.expected, result)
			}
		})
	}
}
