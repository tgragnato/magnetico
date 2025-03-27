package metainfo_test

import (
	"testing"

	"tgragnato.it/magnetico/v2/metainfo"
)

func TestFileInfo_DisplayPath(t *testing.T) {
	t.Parallel()

	info := &metainfo.Info{
		Pieces:      make([]byte, 20),
		PieceLength: 1,
		Length:      20,
		Files:       []metainfo.FileInfo{{Length: 1, Path: []string{"file1"}}},
	}

	fi := &metainfo.FileInfo{
		PathUtf8: []string{"dir1", "dir2", "file.txt"},
		Path:     []string{"dir1", "dir2", "file.txt"},
	}

	expected := "dir1/dir2/file.txt"
	result := fi.DisplayPath(info)

	if result != expected {
		t.Errorf("Expected display path '%s', but got '%s'", expected, result)
	}

	info = &metainfo.Info{
		Pieces:      make([]byte, 20),
		PieceLength: 1,
		Length:      20,
		Name:        "file.txt",
		Files:       []metainfo.FileInfo{},
	}

	result = fi.DisplayPath(info)
	expected = "file.txt"

	if result != expected {
		t.Errorf("Expected display path '%s', but got '%s'", expected, result)
	}
}
