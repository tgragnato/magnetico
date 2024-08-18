package metainfo

import "testing"

func TestChoosePieceLength(t *testing.T) {
	t.Parallel()

	for totalLength := int64(0); totalLength <= 4294967296; totalLength += 1024 {
		pieceLength := ChoosePieceLength(totalLength)
		if pieceLength%minimumPieceLength != 0 {
			t.Errorf("Piece length %d is not a multiple of %d", pieceLength, minimumPieceLength)
		}
		if totalLength/pieceLength >= 2048 {
			t.Errorf("Piece length %d is too small for total length %d", pieceLength, totalLength)
		}
	}
}
