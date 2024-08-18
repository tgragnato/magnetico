package metainfo

import "github.com/tgragnato/magnetico/types/infohash"

// Uniquely identifies a piece.
type PieceKey struct {
	InfoHash infohash.T
	Index    pieceIndex
}
