package metainfo

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"tgragnato.it/magnetico/v2/merkle"
)

func sharedLayerTree(n int, root [32]byte) FileTree {
	ft := FileTree{Dir: map[string]FileTree{}}
	for i := range n {
		ft.Dir[fmt.Sprintf("f%d", i)] = FileTree{
			File: FileTreeFile{Length: 1 << 20, PiecesRoot: string(root[:])},
		}
	}
	return ft
}

func TestValidatePieceLayersSharedRoot(t *testing.T) {
	t.Parallel()

	const pieceLength = 1 << 14
	h1, h2, h3 := sha256.Sum256([]byte("a")), sha256.Sum256([]byte("b")), sha256.Sum256([]byte("c"))
	layer := string(h1[:]) + string(h2[:]) + string(h3[:])
	hashes, _ := merkle.CompactLayerToSliceHashes(layer)
	root := merkle.RootWithPadHash(hashes, HashForPiecePad(pieceLength))

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		ft := sharedLayerTree(100, root)
		if err := ValidatePieceLayers(map[string]string{string(root[:]): layer}, &ft, pieceLength); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("mismatching layer", func(t *testing.T) {
		t.Parallel()
		ft := sharedLayerTree(100, root)
		bad := string(h2[:]) + string(h1[:]) + string(h3[:])
		if err := ValidatePieceLayers(map[string]string{string(root[:]): bad}, &ft, pieceLength); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("missing layer", func(t *testing.T) {
		t.Parallel()
		ft := sharedLayerTree(3, root)
		if err := ValidatePieceLayers(map[string]string{}, &ft, pieceLength); err == nil {
			t.Fatal("expected error")
		}
	})
}
