package merkle

import (
	"crypto/sha256"
	"hash"
)

func NewHash() *Hash {
	h := &Hash{
		nextBlock: sha256.New(),
	}
	return h
}

type Hash struct {
	blocks    [][sha256.Size]byte
	nextBlock hash.Hash
	// How many bytes have been written to nextBlock so far.
	nextBlockWritten int
}

func (h *Hash) remaining() int {
	return BlockSize - h.nextBlockWritten
}

func (h *Hash) Write(p []byte) (n int, err error) {
	for len(p) > 0 {
		var n1 int
		n1, err = h.nextBlock.Write(p[:min(len(p), h.remaining())])
		n += n1
		h.nextBlockWritten += n1
		p = p[n1:]
		if h.remaining() == 0 {
			h.blocks = append(h.blocks, h.nextBlockSum())
			h.nextBlock.Reset()
			h.nextBlockWritten = 0
		}
		if err != nil {
			break
		}
	}
	return
}

func (h *Hash) nextBlockSum() (sum [sha256.Size]byte) {
	copy(sum[:], h.nextBlock.Sum(sum[:0]))
	return
}

func (h *Hash) curBlocks() [][sha256.Size]byte {
	blocks := h.blocks
	if h.nextBlockWritten != 0 {
		blocks = append(blocks, h.nextBlockSum())
	}
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
	blocks = append(blocks, make([][sha256.Size]byte, minBlocks-len(blocks))...)
	sum := RootWithPadHash(blocks, [sha256.Size]byte{})
	return append(b, sum[:]...)
}

// Reset resets the Hash to its initial state.
func (h *Hash) Reset() {
	h.blocks = h.blocks[:0]
	h.nextBlock.Reset()
	h.nextBlockWritten = 0
}

// Size returns the size of the hash in bytes.
func (h *Hash) Size() int {
	return sha256.Size
}

// BlockSize returns the block size of the hash.
func (h *Hash) BlockSize() int {
	return h.nextBlock.BlockSize()
}

var _ hash.Hash = (*Hash)(nil)
