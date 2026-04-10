package merkle

import (
	"crypto/sha256"
	"hash"
)

func NewHash() *Hash {
	return &Hash{}
}

type Hash struct {
	blocks    [][sha256.Size]byte
	nextBlock [BlockSize]byte
	// How many bytes have been written to nextBlock so far.
	nextBlockWritten int
}

func (h *Hash) Write(p []byte) (n int, err error) {
	if h.nextBlockWritten != 0 {
		n1 := copy(h.nextBlock[h.nextBlockWritten:], p)
		h.nextBlockWritten += n1
		n += n1
		p = p[n1:]
		if h.nextBlockWritten == BlockSize {
			h.blocks = append(h.blocks, sha256.Sum256(h.nextBlock[:]))
			h.nextBlockWritten = 0
		}
	}

	for len(p) >= BlockSize {
		h.blocks = append(h.blocks, sha256.Sum256(p[:BlockSize]))
		p = p[BlockSize:]
		n += BlockSize
	}

	if len(p) != 0 {
		n1 := copy(h.nextBlock[:], p)
		h.nextBlockWritten = n1
		n += n1
	}
	return
}

func (h *Hash) nextBlockSum() (sum [sha256.Size]byte) {
	if h.nextBlockWritten == 0 {
		return
	}
	return sha256.Sum256(h.nextBlock[:h.nextBlockWritten])
}

func (h *Hash) curBlocks() [][sha256.Size]byte {
	if h.nextBlockWritten == 0 {
		return h.blocks
	}
	blocks := make([][sha256.Size]byte, len(h.blocks)+1)
	copy(blocks, h.blocks)
	blocks[len(h.blocks)] = h.nextBlockSum()
	return blocks
}

func (h *Hash) Sum(b []byte) []byte {
	sum := RootWithPadHash(h.curBlocks(), [sha256.Size]byte{})
	return append(b, sum[:]...)
}

// Sums by extending with zero hashes for blocks missing to meet the given length. Necessary for
// piece layers hashes for file tail blocks that don't pad to the piece length.
func (h *Hash) SumMinLength(b []byte, length int) []byte {
	blocks := h.curBlocks()
	minBlocks := (length + BlockSize - 1) / BlockSize
	if minBlocks > len(blocks) {
		padded := make([][sha256.Size]byte, minBlocks)
		copy(padded, blocks)
		blocks = padded
	}
	sum := RootWithPadHash(blocks, [sha256.Size]byte{})
	return append(b, sum[:]...)
}

// Reset resets the Hash to its initial state.
func (h *Hash) Reset() {
	h.blocks = h.blocks[:0]
	h.nextBlockWritten = 0
}

// Size returns the size of the hash in bytes.
func (h *Hash) Size() int {
	return sha256.Size
}

// BlockSize returns the block size of the hash.
func (h *Hash) BlockSize() int {
	return sha256.BlockSize
}

var _ hash.Hash = (*Hash)(nil)
