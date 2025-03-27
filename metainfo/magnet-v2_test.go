package metainfo

import (
	"net/url"
	"testing"

	"tgragnato.it/magnetico/v2/types/infohash"
	infohash_v2 "tgragnato.it/magnetico/v2/types/infohash-v2"
)

func TestParseMagnetV2(t *testing.T) {
	t.Parallel()

	const v2Only = "magnet:?xt=urn:btmh:1220caf1e1c30e81cb361b9ee167c4aa64228a7fa4fa9f6105232b28ad099f3a302e&dn=bittorrent-v2-test"

	m2, err := ParseMagnetV2Uri(v2Only)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if m2.InfoHash.HexString() != "0000000000000000000000000000000000000000" {
		t.Errorf("unexpected InfoHash: %s", m2.InfoHash.HexString())
	}
	if m2.V2InfoHash.HexString() != "caf1e1c30e81cb361b9ee167c4aa64228a7fa4fa9f6105232b28ad099f3a302e" {
		t.Errorf("unexpected V2InfoHash: %s", m2.V2InfoHash.HexString())
	}
	if len(m2.Params) != 0 {
		t.Errorf("unexpected Params length: %d", len(m2.Params))
	}

	_, err = ParseMagnetUri(v2Only)
	if err == nil {
		t.Errorf("missing expected error")
	}
	if err.Error() != "missing v1 infohash" {
		t.Errorf("unexpected error: %v", err)
	}

	const hybrid = "magnet:?xt=urn:btih:631a31dd0a46257d5078c0dee4e66e26f73e42ac&xt=urn:btmh:1220d8dd32ac93357c368556af3ac1d95c9d76bd0dff6fa9833ecdac3d53134efabb&dn=bittorrent-v1-v2-hybrid-test"

	m2, err = ParseMagnetV2Uri(hybrid)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if m2.InfoHash.HexString() != "631a31dd0a46257d5078c0dee4e66e26f73e42ac" {
		t.Errorf("unexpected InfoHash: %s", m2.InfoHash.HexString())
	}
	if m2.V2InfoHash.HexString() != "d8dd32ac93357c368556af3ac1d95c9d76bd0dff6fa9833ecdac3d53134efabb" {
		t.Errorf("unexpected V2InfoHash: %s", m2.V2InfoHash.HexString())
	}
	if len(m2.Params) != 0 {
		t.Errorf("unexpected Params length: %d", len(m2.Params))
	}

	m, err := ParseMagnetUri(hybrid)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if m.InfoHash.HexString() != "631a31dd0a46257d5078c0dee4e66e26f73e42ac" {
		t.Errorf("unexpected InfoHash: %s", m.InfoHash.HexString())
	}
	if len(m.Params["xt"]) != 1 {
		t.Errorf("unexpected Params length: %d", len(m.Params["xt"]))
	}
}

func TestMagnetV2String(t *testing.T) {
	t.Parallel()

	m := MagnetV2{
		InfoHash:    infohash.T{},
		V2InfoHash:  infohash_v2.T{},
		Trackers:    []string{"http://tracker1.example.com", "http://tracker2.example.com"},
		DisplayName: "Test Magnet",
		Params: url.Values{
			"param1": []string{"value1"},
			"param2": []string{"value2"},
		},
	}

	expected := "magnet:?dn=Test+Magnet&param1=value1&param2=value2&tr=http%3A%2F%2Ftracker1.example.com&tr=http%3A%2F%2Ftracker2.example.com"
	actual := m.String()
	if actual != expected {
		t.Errorf("unexpected result: got %s, want %s", actual, expected)
	}
}
