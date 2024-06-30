package torrent

import (
	"os"
	"testing"

	_ "github.com/anacrolix/envpprof"
)

func TestMain(m *testing.M) {
	code := m.Run()
	// select {}
	os.Exit(code)
}
