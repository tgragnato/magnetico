package metainfo

import (
	"fmt"
	"reflect"

	"tgragnato.it/magnetico/v2/merkle"
)

func ValidatePieceLayers(
	pieceLayers map[string]string,
	fileTree *FileTree,
	pieceLength int64,
) (err error) {
	var (
		pad       [32]byte
		padReady  bool
		validated = make(map[[32]byte]struct{})
	)
	fileTree.Walk(nil, func(path []string, ft *FileTree) {
		if err != nil {
			return
		}
		if ft.IsDir() {
			return
		}
		piecesRoot := ft.PiecesRootAsByteArray()
		if reflect.DeepEqual(piecesRoot, [32]byte{}) {
			return
		}
		if _, done := validated[piecesRoot]; done {
			return
		}
		filePieceLayers, ok := pieceLayers[string(piecesRoot[:])]
		if !ok {
			// BEP 52: "For each file in the file tree that is larger than the piece size it
			// contains one string value.". The reference torrent creator in
			// https://blog.libtorrent.org/2020/09/bittorrent-v2/ also has this. If a file is equal
			// to or smaller than the piece length, we can just use the pieces root instead of the
			// piece layer hash.
			if ft.File.Length > pieceLength {
				err = fmt.Errorf("no piece layers for file %q", path)
			}
			return
		}
		var layerHashes [][32]byte
		layerHashes, err = merkle.CompactLayerToSliceHashes(filePieceLayers)
		if !padReady {
			pad, padReady = HashForPiecePad(pieceLength), true
		}
		root := merkle.RootWithPadHash(layerHashes, pad)
		if root != piecesRoot {
			err = fmt.Errorf("file %q: expected hash %x got %x", path, piecesRoot, root)
			return
		}
		validated[piecesRoot] = struct{}{}
	})
	return
}

// Returns the padding hash for the hash layer corresponding to a piece. It can't be zero because
// that's the bottom-most layer (the hashes for the smallest blocks).
func HashForPiecePad(pieceLength int64) (hash [32]byte) {
	// This should be a power of two, and probably checked elsewhere.
	blocksPerPiece := pieceLength / (1 << 14)
	blockHashes := make([][32]byte, blocksPerPiece)
	return merkle.Root(blockHashes)
}
