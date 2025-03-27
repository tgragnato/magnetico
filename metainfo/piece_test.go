package metainfo

import (
	"testing"

	"tgragnato.it/magnetico/v2/types/infohash"
)

func TestPiece_Length(t *testing.T) {
	t.Parallel()

	info := &Info{
		PieceLength: 1024,
		Files:       []FileInfo{{Length: 2048}},
		Pieces:      make([]byte, 4096),
	}
	piece := Piece{
		Info:  info,
		index: 0,
	}

	// Test case 1: V2 piece length calculation
	expectedLength := int64(1024)
	actualLength := piece.Length()
	if actualLength != expectedLength {
		t.Errorf("Expected length: %d, but got: %d", expectedLength, actualLength)
	}

	// Test case 2: V1 piece length calculation
	info.Files = []FileInfo{{Length: 3072, TorrentOffset: 0}}
	piece.index = 2
	expectedLength = int64(1024)
	actualLength = piece.Length()
	if actualLength != expectedLength {
		t.Errorf("Expected length: %d, but got: %d", expectedLength, actualLength)
	}

	// Test case 3: V1 piece length calculation for last piece
	info.Files = []FileInfo{
		{Length: 2048, TorrentOffset: 0},
		{Length: 1024, TorrentOffset: 2048},
	}
	piece.index = 2
	expectedLength = int64(1024)
	actualLength = piece.Length()
	if actualLength != expectedLength {
		t.Errorf("Expected length: %d, but got: %d", expectedLength, actualLength)
	}

	// Test case 4: V1 piece length calculation for single file
	info.Files = []FileInfo{{Length: 4096, TorrentOffset: 0}}
	piece.index = 1
	expectedLength = int64(1024)
	actualLength = piece.Length()
	if actualLength != expectedLength {
		t.Errorf("Expected length: %d, but got: %d", expectedLength, actualLength)
	}
}

func TestPiece_Offset(t *testing.T) {
	t.Parallel()

	info := &Info{
		PieceLength: 1024,
		Files:       []FileInfo{{Length: 2048}},
		Pieces:      make([]byte, 4096),
	}
	piece := Piece{
		Info:  info,
		index: 0,
	}

	// Test case 1: Offset calculation for index 0
	expectedOffset := int64(0)
	actualOffset := piece.Offset()
	if actualOffset != expectedOffset {
		t.Errorf("Expected offset: %d, but got: %d", expectedOffset, actualOffset)
	}

	// Test case 2: Offset calculation for index 1
	piece.index = 1
	expectedOffset = int64(1024)
	actualOffset = piece.Offset()
	if actualOffset != expectedOffset {
		t.Errorf("Expected offset: %d, but got: %d", expectedOffset, actualOffset)
	}

	// Test case 3: Offset calculation for index 2
	piece.index = 2
	expectedOffset = int64(2048)
	actualOffset = piece.Offset()
	if actualOffset != expectedOffset {
		t.Errorf("Expected offset: %d, but got: %d", expectedOffset, actualOffset)
	}
}

func TestPiece_V1Hash(t *testing.T) {
	t.Parallel()

	info := &Info{
		PieceLength: 1024,
		Pieces:      make([]byte, 4096),
	}
	piece := Piece{
		Info:  info,
		index: 0,
	}

	// Test case 1: V1Hash calculation for index 0
	expectedHash := infohash.T{}
	actualHash := piece.V1Hash()
	if actualHash != expectedHash {
		t.Errorf("Expected hash: %v, but got: %v", expectedHash, actualHash)
	}

	// Test case 2: V1Hash calculation for index 1
	piece.index = 1
	expectedHash = infohash.T{}
	actualHash = piece.V1Hash()
	if actualHash != expectedHash {
		t.Errorf("Expected hash: %v, but got: %v", expectedHash, actualHash)
	}

	// Test case 3: V1Hash calculation for index 2
	piece.index = 2
	expectedHash = infohash.T{}
	actualHash = piece.V1Hash()
	if actualHash != expectedHash {
		t.Errorf("Expected hash: %v, but got: %v", expectedHash, actualHash)
	}
}
