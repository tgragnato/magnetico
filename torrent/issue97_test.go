package torrent

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/anacrolix/torrent/internal/testutil"
	"github.com/anacrolix/torrent/storage"
)

func TestHashPieceAfterStorageClosed(t *testing.T) {
	td := t.TempDir()
	cs := storage.NewFile(td)
	defer cs.Close()
	tt := &Torrent{
		storageOpener: storage.NewClient(cs),
		chunkSize:     defaultChunkSize,
	}
	tt.infoHash.Ok = true
	tt.infoHash.Value[0] = 1
	mi := testutil.GreetingMetaInfo()
	info, err := mi.UnmarshalInfo()
	require.NoError(t, err)
	require.NoError(t, tt.setInfo(&info))
	require.NoError(t, tt.storage.Close())
	tt.hashPiece(0)
}
