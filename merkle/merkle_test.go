package merkle

import (
	"crypto/sha256"
	"testing"
)

func TestRoot_EmptyHashes(t *testing.T) {
	t.Parallel()

	hashes := [][sha256.Size]byte{}
	expectedRoot := sha256.Sum256(nil)

	root := Root(hashes)

	if root != expectedRoot {
		t.Errorf("Expected root %x, but got %x", expectedRoot, root)
	}
}

func TestRoot_SingleHash(t *testing.T) {
	t.Parallel()

	hash := sha256.Sum256([]byte("Hello, World!"))
	hashes := [][sha256.Size]byte{hash}
	expectedRoot := hash

	root := Root(hashes)

	if root != expectedRoot {
		t.Errorf("Expected root %x, but got %x", expectedRoot, root)
	}
}

func TestRoot_MultipleHashes(t *testing.T) {
	t.Parallel()

	hash1 := sha256.Sum256([]byte("Hello"))
	hash2 := sha256.Sum256([]byte("World"))
	hashes := [][sha256.Size]byte{hash1, hash2}
	expectedRoot := sha256.Sum256(append(hash1[:], hash2[:]...))

	root := Root(hashes)

	if root != expectedRoot {
		t.Errorf("Expected root %x, but got %x", expectedRoot, root)
	}
}

func TestLog2RoundingUp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		n        uint
		expected uint
	}{
		{0, 0},
		{1, 0},
		{2, 1},
		{3, 2},
		{4, 2},
		{5, 3},
		{8, 3},
		{9, 4},
		{16, 4},
		{17, 5},
		{32, 5},
		{33, 6},
		{64, 6},
		{65, 7},
		{128, 7},
		{129, 8},
		{256, 8},
		{257, 9},
	}

	for _, test := range tests {
		result := Log2RoundingUp(test.n)
		if result != test.expected {
			t.Errorf("For n=%d, expected %d, but got %d", test.n, test.expected, result)
		}
	}
}

func TestRoundUpToPowerOfTwo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		n        uint
		expected uint
	}{
		{0, 0},
		{1, 1},
		{2, 2},
		{3, 4},
		{4, 4},
		{5, 8},
		{8, 8},
		{9, 16},
		{16, 16},
		{17, 32},
		{32, 32},
		{33, 64},
		{64, 64},
		{65, 128},
		{128, 128},
		{129, 256},
		{256, 256},
		{257, 512},
	}

	for _, test := range tests {
		result := RoundUpToPowerOfTwo(test.n)
		if result != test.expected {
			t.Errorf("For n=%d, expected %d, but got %d", test.n, test.expected, result)
		}
	}
}
