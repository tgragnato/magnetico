package merkle

import (
	"crypto/sha256"
	"testing"
)

func TestHash_BlockSize(t *testing.T) {
	t.Parallel()

	h := NewHash()
	expectedBlockSize := sha256.BlockSize

	if blockSize := h.BlockSize(); blockSize != expectedBlockSize {
		t.Errorf("Expected block size %d, but got %d", expectedBlockSize, blockSize)
	}
}

func TestHash_Write_Reset(t *testing.T) {
	t.Parallel()

	h := NewHash()
	data := []byte("Hello, World!")

	n, err := h.Write(data)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if n != len(data) {
		t.Errorf("Expected to write %d bytes, but wrote %d bytes", len(data), n)
	}

	h.Reset()
	if len(h.blocks) != 0 {
		t.Errorf("Expected blocks length 0 after reset, but got %d", len(h.blocks))
	}

	if h.nextBlockWritten != 0 {
		t.Errorf("Expected nextBlockWritten to be 0 after reset, but got %d", h.nextBlockWritten)
	}
}

func TestHash_Sum(t *testing.T) {
	t.Parallel()

	h := NewHash()
	data := []byte("Hello, World!")

	_, err := h.Write(data)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expectedSum := sha256.Sum256(data)
	sum := h.Sum(nil)

	if len(sum) != len(expectedSum) {
		t.Errorf("Expected sum length %d, but got %d", len(expectedSum), len(sum))
	}

	for i := 0; i < len(sum); i++ {
		if sum[i] != expectedSum[i] {
			t.Errorf("Expected sum %x, but got %x", expectedSum, sum)
			break
		}
	}
}

func TestHash_Size(t *testing.T) {
	t.Parallel()

	h := NewHash()
	expectedSize := 32

	if size := h.Size(); size != expectedSize {
		t.Errorf("Expected hash size %d, but got %d", expectedSize, size)
	}
}

func TestHash_SumMinLength(t *testing.T) {
	t.Parallel()

	h := NewHash()
	data := []byte("Hello, World!")
	expectedLength := 45

	sum := h.SumMinLength(data, 20+len(data))
	if len(sum) != expectedLength {
		t.Errorf("Expected sum length %d, but got %d", expectedLength, len(sum))
	}
}
