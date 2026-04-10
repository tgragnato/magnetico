package merkle

import (
	"crypto/sha256"
	"fmt"
	"reflect"
	"slices"
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

func TestCompactLayerToSliceHashes_Sample(t *testing.T) {
	t.Parallel()

	tests := []struct {
		compactLayer string
		expected     [][sha256.Size]byte
	}{
		{
			compactLayer: "",
			expected:     [][sha256.Size]byte{},
		},
		{
			compactLayer: "0123456789abcdef",
			expected:     [][sha256.Size]byte{},
		},
		{
			compactLayer: "0123456789abcdef0123456789abcdef",
			expected:     [][sha256.Size]byte{{48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 97, 98, 99, 100, 101, 102, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 97, 98, 99, 100, 101, 102}},
		},
		{
			compactLayer: "0123456789abcdef0123456789abcdef0123456789abcdef",
			expected:     [][sha256.Size]byte{{48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 97, 98, 99, 100, 101, 102, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 97, 98, 99, 100, 101, 102}},
		},
	}

	for _, test := range tests {
		result, err := CompactLayerToSliceHashes(test.compactLayer)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("For compactLayer=%q, expected %v, but got %v", test.compactLayer, test.expected, result)
		}
	}
}

func TestRootMatchesReference(t *testing.T) {
	t.Parallel()
	for _, leaves := range []int{0, 1, 2, 4, 8, 64, 1024} {
		t.Run(fmt.Sprintf("leaves=%d", leaves), func(t *testing.T) {
			t.Parallel()
			hashes := makeTestHashes(leaves)
			got := Root(hashes)
			want := referenceRoot(hashes)
			if got != want {
				t.Fatalf("got %x, want %x", got, want)
			}
		})
	}
}

func TestRootWithPadHashMatchesReference(t *testing.T) {
	t.Parallel()
	padHash := sha256.Sum256([]byte("pad"))
	for _, leaves := range []int{0, 1, 3, 5, 63, 511} {
		t.Run(fmt.Sprintf("leaves=%d", leaves), func(t *testing.T) {
			t.Parallel()
			hashes := makeTestHashes(leaves)
			got := RootWithPadHash(hashes, padHash)
			want := referenceRootWithPadHash(hashes, padHash)
			if got != want {
				t.Fatalf("got %x, want %x", got, want)
			}
		})
	}
}

func TestRootDoesNotMutateInput(t *testing.T) {
	t.Parallel()
	hashes := makeTestHashes(8)
	original := append([][sha256.Size]byte(nil), hashes...)
	_ = Root(hashes)
	if len(hashes) != len(original) {
		t.Fatalf("input length changed from %d to %d", len(original), len(hashes))
	}
	for i := range hashes {
		if hashes[i] != original[i] {
			t.Fatalf("input hash at index %d changed", i)
		}
	}
}

func TestRootPanicsForNonPowerOfTwo(t *testing.T) {
	t.Parallel()
	hashes := makeTestHashes(3)
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for non-power-of-two hash count")
		}
	}()
	_ = Root(hashes)
}

func TestCompactLayerToSliceHashes(t *testing.T) {
	t.Parallel()
	hashes := makeTestHashes(4)
	var compact []byte
	for _, hash := range hashes {
		compact = append(compact, hash[:]...)
	}
	got, err := CompactLayerToSliceHashes(string(compact))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slices.Equal(got, hashes) {
		t.Fatalf("got %x, want %x", got, hashes)
	}
}

func BenchmarkRoot(b *testing.B) {
	hashes := makeTestHashes(1024)
	b.ReportAllocs()
	for b.Loop() {
		_ = Root(hashes)
	}
}

func BenchmarkRootWithPadHash(b *testing.B) {
	hashes := makeTestHashes(511)
	padHash := sha256.Sum256([]byte("pad"))
	b.ReportAllocs()
	for b.Loop() {
		_ = RootWithPadHash(hashes, padHash)
	}
}

func makeTestHashes(n int) [][sha256.Size]byte {
	hashes := make([][sha256.Size]byte, n)
	for i := range hashes {
		hashes[i] = sha256.Sum256([]byte(fmt.Sprintf("leaf-%d", i)))
	}
	return hashes
}

func referenceRoot(hashes [][sha256.Size]byte) [sha256.Size]byte {
	switch len(hashes) {
	case 0:
		return sha256.Sum256(nil)
	case 1:
		return hashes[0]
	}
	numHashes := uint(len(hashes))
	if numHashes != RoundUpToPowerOfTwo(uint(len(hashes))) {
		panic(fmt.Sprintf("expected power of two number of hashes, got %d", numHashes))
	}
	var next [][sha256.Size]byte
	for i := 0; i < len(hashes); i += 2 {
		left := hashes[i]
		right := hashes[i+1]
		next = append(next, sha256.Sum256(append(left[:], right[:]...)))
	}
	return referenceRoot(next)
}

func referenceRootWithPadHash(hashes [][sha256.Size]byte, padHash [sha256.Size]byte) [sha256.Size]byte {
	for uint(len(hashes)) < RoundUpToPowerOfTwo(uint(len(hashes))) {
		hashes = append(hashes, padHash)
	}
	return referenceRoot(hashes)
}
