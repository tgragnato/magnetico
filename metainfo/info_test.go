package metainfo

import (
	"testing"

	"tgragnato.it/magnetico/v2/bencode"
)

func TestMarshalInfo(t *testing.T) {
	t.Parallel()

	var info Info
	info.Pieces = make([]byte, 0)
	b, err := bencode.Marshal(info)
	if err != nil {
		t.Errorf("Error marshaling info: %v", err)
	}
	expected := "d4:name0:12:piece lengthi0e6:pieces0:e"
	if string(b) != expected {
		t.Errorf("Unexpected marshaled value. Expected: %s, Got: %s", expected, string(b))
	}
}

func TestTotalLength(t *testing.T) {
	t.Parallel()

	info := Info{
		Files: []FileInfo{
			{Length: 100},
			{Length: 200},
			{Length: 300},
		},
	}

	expected := int64(600)
	actual := info.TotalLength()

	if expected != actual {
		t.Errorf("Unexpected total length. Expected: %d, Got: %d", expected, actual)
	}
}

func TestIsDir(t *testing.T) {
	t.Parallel()

	// Test when Info.Files is empty
	info := Info{}
	expected := false
	actual := info.IsDir()
	if expected != actual {
		t.Errorf("Unexpected result. Expected: %v, Got: %v", expected, actual)
	}

	// Test when Info.Files is not empty
	info = Info{
		Files: []FileInfo{
			{Length: 100},
			{Length: 200},
			{Length: 300},
		},
	}
	expected = true
	actual = info.IsDir()
	if expected != actual {
		t.Errorf("Unexpected result. Expected: %v, Got: %v", expected, actual)
	}
}
