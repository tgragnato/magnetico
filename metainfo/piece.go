package metainfo

import (
	"iter"

	"tgragnato.it/magnetico/v2/types/infohash"
)

type Piece struct {
	Info  *Info
	index int
}

func (p Piece) Length() int64 {
	if p.Info.HasV2() {
		var offset int64
		pieceLength := p.Info.PieceLength
		lastFileEnd := int64(0)
		for fi := range p.Info.FileTree.upvertedFiles(pieceLength) {
			fileStartPiece := int(offset / pieceLength)
			if fileStartPiece > p.index {
				break
			}
			lastFileEnd = offset + fi.Length
			offset = (lastFileEnd + pieceLength - 1) / pieceLength * pieceLength

		}
		ret := min(lastFileEnd-int64(p.index)*pieceLength, pieceLength)
		if ret <= 0 {
			return 0
		}
		return ret
	}
	return p.V1Length()
}

func iterLast[T any](i iter.Seq[T]) (last T, ok bool) {
	for t := range i {
		last = t
		ok = true
	}
	return
}

func (p Piece) V1Length() int64 {
	i := p.index
	lastPiece := p.Info.NumPieces() - 1
	switch {
	case 0 <= i && i < lastPiece:
		return p.Info.PieceLength
	case lastPiece >= 0 && i == lastPiece:
		lastFile, ok := iterLast(p.Info.UpvertedV1Files())
		if !ok {
			return 0
		}
		length := lastFile.TorrentOffset + lastFile.Length - int64(i)*p.Info.PieceLength
		if length <= 0 || length > p.Info.PieceLength {
			return 0
		}
		return length
	default:
		return 0
	}
}

func (p Piece) Offset() int64 {
	return int64(p.index) * p.Info.PieceLength
}

func (p Piece) V1Hash() (ret infohash.T) {
	if !p.Info.HasV1() {
		return infohash.T{}
	}
	copy(ret[:], p.Info.Pieces[p.index*infohash.Size:(p.index+1)*infohash.Size])
	return
}

func (p Piece) Index() int {
	return p.index
}
