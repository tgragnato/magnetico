package metainfo

import (
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tgragnato.it/magnetico/v2/bencode"
)

func testFile(t *testing.T, filename string) {
	mi, err := LoadFromFile(filename)
	if err != nil {
		t.Errorf("Error loading file: %v", err)
		return
	}
	info, err := mi.UnmarshalInfo()
	if err != nil {
		t.Errorf("Error unmarshaling info: %v", err)
		return
	}

	if len(info.Files) == 1 {
		t.Logf("Single file: %s (length: %d)\n", info.BestName(), info.Files[0].Length)
	} else {
		t.Logf("Multiple files: %s\n", info.BestName())
		for _, f := range info.Files {
			t.Logf(" - %s (length: %d)\n", path.Join(f.Path...), f.Length)
		}
	}

	for _, group := range mi.AnnounceList {
		for _, tracker := range group {
			t.Logf("Tracker: %s\n", tracker)
		}
	}

	b, err := bencode.Marshal(&info)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if string(b) != string(mi.InfoBytes) {
		t.Errorf("Expected %s, but got %s", string(mi.InfoBytes), string(b))
	}
}

func TestFile(t *testing.T) {
	t.Parallel()

	testFile(t, "testdata/archlinux-2011.08.19-netinstall-i686.iso.torrent")
	testFile(t, "testdata/continuum.torrent")
	testFile(t, "testdata/23516C72685E8DB0C8F15553382A927F185C4F01.torrent")
	testFile(t, "testdata/trackerless.torrent")
	_, err := LoadFromFile("testdata/minimal-trailing-newline.torrent")
	if err != nil {
		t.Errorf("Expected EOF error, but got: %v", err)
	}
}

var ZeroReader zeroReader

type zeroReader struct{}

func (me zeroReader) Read(b []byte) (n int, err error) {
	for i := range b {
		b[i] = 0
	}
	n = len(b)
	return
}

// Ensure that the correct number of pieces are generated when hashing files.
func TestNumPieces(t *testing.T) {
	t.Parallel()

	for _, _case := range []struct {
		PieceLength int64
		Files       []FileInfo
		NumPieces   int
	}{
		{256 * 1024, []FileInfo{{Length: 1024*1024 + -1}}, 4},
		{256 * 1024, []FileInfo{{Length: 1024 * 1024}}, 4},
		{256 * 1024, []FileInfo{{Length: 1024*1024 + 1}}, 5},
		{5, []FileInfo{{Length: 1}, {Length: 12}}, 3},
		{5, []FileInfo{{Length: 4}, {Length: 12}}, 4},
	} {
		info := Info{
			Files:       _case.Files,
			PieceLength: _case.PieceLength,
		}
		err := info.GeneratePieces(func(fi FileInfo) (io.ReadCloser, error) {
			return io.NopCloser(ZeroReader), nil
		})
		if err != nil {
			t.Errorf("Error: %v", err)
		} else if info.NumPieces() != _case.NumPieces {
			t.Errorf("Expected %d pieces, but got %d", _case.NumPieces, info.NumPieces())
		}
	}
}

func touchFile(path string) (err error) {
	f, err := os.Create(path)
	if err != nil {
		return
	}
	err = f.Close()
	return
}

func TestBuildFromFilePathOrder(t *testing.T) {
	t.Parallel()

	td := t.TempDir()
	err := touchFile(filepath.Join(td, "b"))
	if err != nil {
		t.Errorf("Error creating file: %v", err)
	}
	err = touchFile(filepath.Join(td, "a"))
	if err != nil {
		t.Errorf("Error creating file: %v", err)
	}
	info := Info{
		PieceLength: 1,
	}
	err = info.BuildFromFilePath(td)
	if err != nil {
		t.Errorf("Error building from file path: %v", err)
	}
	if len(info.Files) != 2 {
		t.Errorf("Expected 2 files, but got %d", len(info.Files))
	} else {
		if info.Files[0].Path[0] != "a" {
			t.Errorf("Expected file path 'a', but got '%s'", info.Files[0].Path[0])
		}
		if info.Files[1].Path[0] != "b" {
			t.Errorf("Expected file path 'b', but got '%s'", info.Files[1].Path[0])
		}
	}
}

func testUnmarshal(t *testing.T, input string, expected *MetaInfo) {
	var actual MetaInfo
	err := bencode.Unmarshal([]byte(input), &actual)
	if expected == nil {
		if err == nil {
			t.Errorf("Expected error, but got nil")
		}
		return
	}
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !reflect.DeepEqual(*expected, actual) {
		t.Errorf("Expected %v, but got %v", *expected, actual)
	}
}

func TestUnmarshal(t *testing.T) {
	t.Parallel()

	testUnmarshal(t, `de`, &MetaInfo{})
	testUnmarshal(t, `d4:infoe`, nil)
	testUnmarshal(t, `d4:infoabce`, nil)
	testUnmarshal(t, `d4:infodee`, &MetaInfo{InfoBytes: []byte("de")})
}

func TestMetainfoWithListURLList(t *testing.T) {
	t.Parallel()

	mi, err := LoadFromFile("testdata/SKODAOCTAVIA336x280_archive.torrent")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(mi.UrlList) != 3 {
		t.Errorf("Expected 3 elements in UrlList, but got %d", len(mi.UrlList))
	}
	magnetURL := strings.Join([]string{
		"magnet:?xt=urn:btih:d4b197dff199aad447a9a352e31528adbbd97922",
		"tr=http%3A%2F%2Fbt1.archive.org%3A6969%2Fannounce",
		"tr=http%3A%2F%2Fbt2.archive.org%3A6969%2Fannounce",
		"ws=https%3A%2F%2Farchive.org%2Fdownload%2F",
		"ws=http%3A%2F%2Fia601600.us.archive.org%2F26%2Fitems%2F",
		"ws=http%3A%2F%2Fia801600.us.archive.org%2F26%2Fitems%2F",
	}, "&")
	if mi.Magnet(nil, nil).String() != magnetURL {
		t.Errorf("Expected %s, but got %s", magnetURL, mi.Magnet(nil, nil).String())
	}
}

func TestMetainfoWithStringURLList(t *testing.T) {
	t.Parallel()

	mi, err := LoadFromFile("testdata/flat-url-list.torrent")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(mi.UrlList) != 1 {
		t.Errorf("Expected 1 element in UrlList, but got %d", len(mi.UrlList))
	}
	magnetURL := strings.Join([]string{
		"magnet:?xt=urn:btih:9da24e606e4ed9c7b91c1772fb5bf98f82bd9687",
		"tr=http%3A%2F%2Fbt1.archive.org%3A6969%2Fannounce",
		"tr=http%3A%2F%2Fbt2.archive.org%3A6969%2Fannounce",
		"ws=https%3A%2F%2Farchive.org%2Fdownload%2F",
	}, "&")
	if mi.Magnet(nil, nil).String() != magnetURL {
		t.Errorf("Expected %s, but got %s", magnetURL, mi.Magnet(nil, nil).String())
	}
}

// The decoder buffer wasn't cleared before starting the next dict item after
// a syntax error on a field with the ignore_unmarshal_type_error tag.
func TestStringCreationDate(t *testing.T) {
	t.Parallel()

	var mi MetaInfo
	err := bencode.Unmarshal([]byte("d13:creation date23:29.03.2018 22:18:14 UTC4:infodee"), &mi)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestUnmarshalEmptyStringNodes(t *testing.T) {
	t.Parallel()

	var mi MetaInfo
	err := bencode.Unmarshal([]byte("d5:nodes0:e"), &mi)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestUnmarshalV2Metainfo(t *testing.T) {
	t.Parallel()

	mi, err := LoadFromFile("testdata/bittorrent-v2-test.torrent")
	if err != nil {
		t.Errorf("Error loading file: %v", err)
		return
	}
	info, err := mi.UnmarshalInfo()
	if err != nil {
		t.Errorf("Error unmarshaling info: %v", err)
		return
	}
	if info.NumPieces() == 0 {
		t.Errorf("Expected non-zero number of pieces, but got 0")
	}
	err = ValidatePieceLayers(mi.PieceLayers, &info.FileTree, info.PieceLength)
	if err != nil {
		t.Errorf("Error validating piece layers: %v", err)
	}
}
