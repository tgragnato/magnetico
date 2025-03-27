package metainfo

import (
	"bytes"
	"testing"

	"tgragnato.it/magnetico/v2/bencode"
)

func testFileNodesMatch(t *testing.T, file string, nodes []Node) {
	mi, err := LoadFromFile(file)
	if err != nil {
		t.Errorf("Error occurred: %v", err)
	}
	if len(nodes) != len(mi.Nodes) {
		t.Errorf("Expected %d nodes, but got %d", len(nodes), len(mi.Nodes))
		return
	}

	for i := range nodes {
		if nodes[i] != mi.Nodes[i] {
			t.Errorf("Expected node %d to be %s, but got %s", i, nodes[i], mi.Nodes[i])
		}
	}
}

func TestNodesListStrings(t *testing.T) {
	t.Parallel()

	testFileNodesMatch(t, "testdata/trackerless.torrent", []Node{
		"udp://tracker.openbittorrent.com:80",
		"udp://tracker.openbittorrent.com:80",
	})
}

func TestNodesListPairsBEP5(t *testing.T) {
	t.Parallel()

	testFileNodesMatch(t, "testdata/issue_65a.torrent", []Node{
		"185.34.3.132:5680",
		"185.34.3.103:12340",
		"94.209.253.165:47232",
		"78.46.103.11:34319",
		"195.154.162.70:55011",
		"185.34.3.137:3732",
	})
	testFileNodesMatch(t, "testdata/issue_65b.torrent", []Node{
		"95.211.203.130:6881",
		"84.72.116.169:6889",
		"204.83.98.77:7000",
		"101.187.175.163:19665",
		"37.187.118.32:6881",
		"83.128.223.71:23865",
	})
}

func testMarshalMetainfo(t *testing.T, expected string, mi *MetaInfo) {
	b, err := bencode.Marshal(*mi)
	if err != nil {
		t.Errorf("Error occurred: %v", err)
	}
	if string(b) != expected {
		t.Errorf("Expected %s, but got %s", expected, string(b))
	}
}

func TestMarshalMetainfoNodes(t *testing.T) {
	t.Parallel()

	testMarshalMetainfo(t, "d4:infodee", &MetaInfo{InfoBytes: []byte("de")})
	testMarshalMetainfo(t, "d4:infod2:hi5:theree5:nodesl12:1.2.3.4:555514:not a hostportee", &MetaInfo{
		Nodes:     []Node{"1.2.3.4:5555", "not a hostport"},
		InfoBytes: []byte("d2:hi5:theree"),
	})
}

func TestUnmarshalBadMetainfoNodes(t *testing.T) {
	t.Parallel()

	var mi MetaInfo
	// Should barf on the integer in the nodes list.
	err := bencode.Unmarshal([]byte("d5:nodesl1:ai42eee"), &mi)
	if err == nil {
		t.Error("Expected error, but got nil")
	}
}

func TestMetainfoEmptyInfoBytes(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	mi := &MetaInfo{
		// Include a non-empty field that comes after "info".
		UrlList: []string{"hello"},
	}
	err := mi.Write(&buf)
	if err != nil {
		t.Errorf("Error occurred: %v", err)
	}
	err = bencode.Unmarshal(buf.Bytes(), &mi)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
}
