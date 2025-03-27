package infohash_v2

import (
	"fmt"
	"testing"

	"tgragnato.it/magnetico/v2/types/infohash"
)

func TestT_Format(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		hash     T
		expected string
	}{
		{
			name:     "Empty hash",
			hash:     T{},
			expected: "0000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			name:     "Non-empty hash",
			hash:     T{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0},
			expected: "123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := fmt.Sprintf("%v", &test.hash)
			if result != test.expected {
				t.Errorf("Unexpected result. Expected: %s, Got: %s", test.expected, result)
			}
		})
	}
}

func TestHashFromHexString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected T
	}{
		{
			name:     "Valid hex string",
			input:    "123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0",
			expected: T{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0},
		},
		{
			name:     "Invalid hex string",
			input:    "invalid",
			expected: T{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testHash := T{}
			err := testHash.FromHexString(test.input)
			if err == nil && test.expected == (T{}) {
				t.Errorf("Expected error: %v", err)
			}
			if testHash != test.expected {
				t.Errorf("Unexpected result. Expected: %v, Got: %v", test.expected, testHash)
			}
		})
	}
}

func TestUnmarshalText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected T
	}{
		{
			name:     "Valid hex string",
			input:    "123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0",
			expected: T{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0},
		},
		{
			name:     "Invalid hex string",
			input:    "invalid",
			expected: T{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h := T{}
			err := h.UnmarshalText([]byte(test.input))
			if err != nil {
				if test.expected != (T{}) {
					t.Errorf("Unexpected error: %v", err)
				}
			}
			if h != test.expected {
				t.Errorf("Unexpected result. Expected: %v, Got: %v", test.expected, h)
			}
		})
	}
}

func TestMarshalText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		hash     T
		expected string
	}{
		{
			name:     "Empty hash",
			hash:     T{},
			expected: "0000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			name:     "Non-empty hash",
			hash:     T{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0},
			expected: "123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.hash.MarshalText()
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if string(result) != test.expected {
				t.Errorf("Unexpected result. Expected: %s, Got: %s", test.expected, string(result))
			}
		})
	}
}

func TestFromHexString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected T
	}{
		{
			name:     "Valid hex string",
			input:    "123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0",
			expected: T{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0},
		},
		{
			name:     "Invalid hex string",
			input:    "invalid",
			expected: T{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h := FromHexString(test.input)
			if h != test.expected {
				t.Errorf("Unexpected result. Expected: %v, Got: %v", test.expected, h)
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
			name:     "Empty input",
			input:    []byte{},
			expected: T{227, 176, 196, 66, 152, 252, 28, 20, 154, 251, 244, 200, 153, 111, 185, 36, 39, 174, 65, 228, 100, 155, 147, 76, 164, 149, 153, 27, 120, 82, 184, 85},
		},
		{
			name:     "Non-empty input",
			input:    []byte{0x61, 0x62, 0x63},
			expected: T{0xba, 0x78, 0x16, 0xbf, 0x8f, 0x01, 0xcf, 0xea, 0x41, 0x41, 0x40, 0xde, 0x5d, 0xae, 0x22, 0x23, 0xb0, 0x03, 0x61, 0xa3, 0x96, 0x17, 0x7a, 0x9c, 0xb4, 0x10, 0xff, 0x61, 0xf2, 0x00, 0x15, 0xad},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := HashBytes(test.input)
			if result != test.expected {
				t.Errorf("Unexpected result. Expected: %v, Got: %v", test.expected, result)
			}
		})
	}
}

func TestIsZero(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		hash     T
		expected bool
	}{
		{
			name:     "Empty hash",
			hash:     T{},
			expected: true,
		},
		{
			name:     "Non-empty hash",
			hash:     T{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.hash.IsZero()
			if result != test.expected {
				t.Errorf("Unexpected result. Expected: %v, Got: %v", test.expected, result)
			}
		})
	}
}

func TestToShort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		hash     T
		expected infohash.T
	}{
		{
			name:     "Empty hash",
			hash:     T{},
			expected: infohash.T{},
		},
		{
			name:     "Non-empty hash",
			hash:     T{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0},
			expected: infohash.T{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.hash.ToShort()
			if result != test.expected {
				t.Errorf("Unexpected result. Expected: %v, Got: %v", test.expected, result)
			}
		})
	}
}

func TestToMultihash(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		hash     T
		expected string
	}{
		{
			name:     "Empty hash",
			hash:     T{},
			expected: "QmNLei78zWmzUdbeRB3CiUfAizWUrbeeZh5K1rhAQKCh51",
		},
		{
			name:     "Non-empty hash",
			hash:     T{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0},
			expected: "QmPZiLSCWAZoDCtj5ivmabMfKv7tFP6zn64CS7kh5goaMd",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := ToMultihash(test.hash)
			if result.B58String() != test.expected {
				t.Errorf("Unexpected result. Expected: %s, Got: %s", test.expected, result.B58String())
			}
		})
	}
}
