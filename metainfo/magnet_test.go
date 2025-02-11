package metainfo

import (
	"encoding/hex"
	"reflect"
	"testing"
)

var (
	exampleMagnetURI = `magnet:?xt=urn:btih:51340689c960f0778a4387aef9b4b52fd08390cd&dn=Shit+Movie+%281985%29+1337p+-+Eru&tr=http%3A%2F%2Fhttp.was.great%21&tr=udp%3A%2F%2Fanti.piracy.honeypot%3A6969`
	exampleMagnet    = Magnet{
		DisplayName: "Shit Movie (1985) 1337p - Eru",
		Trackers: []string{
			"http://http.was.great!",
			"udp://anti.piracy.honeypot:6969",
		},
	}
)

func init() {
	_, err := hex.Decode(exampleMagnet.InfoHash[:], []byte("51340689c960f0778a4387aef9b4b52fd08390cd"))
	if err != nil {
		panic(err)
	}
}

// Converting from our Magnet type to URL string.
func TestMagnetString(t *testing.T) {
	t.Parallel()

	m, err := ParseMagnetUri(exampleMagnet.String())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(exampleMagnet, m) {
		t.Errorf("Expected %v, but got %v", exampleMagnet, m)
	}
}

func TestParseMagnetURI(t *testing.T) {
	t.Parallel()

	var uri string
	var m Magnet
	var err error

	// parsing the legit Magnet URI with btih-formatted xt should not return errors
	uri = "magnet:?xt=urn:btih:ZOCMZQIPFFW7OLLMIC5HUB6BPCSDEOQU"
	_, err = ParseMagnetUri(uri)
	if err != nil {
		t.Errorf("Attempting parsing the proper Magnet btih URI:\"%v\" failed with err: %v", uri, err)
	}

	// Checking if the magnet instance struct is built correctly from parsing
	m, err = ParseMagnetUri(exampleMagnetURI)
	if !reflect.DeepEqual(exampleMagnet, m) {
		t.Errorf("Expected %v, but got %v", exampleMagnet, m)
	}
	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}

	// empty string URI case
	_, err = ParseMagnetUri("")
	if err == nil {
		t.Errorf("Parsing empty string as URI should have returned an error but didn't")
	}

	// only BTIH (BitTorrent info hash)-formatted magnet links are currently supported
	// must return error correctly when encountering other URN formats
	uri = "magnet:?xt=urn:sha1:YNCKHTQCWBTRNJIV4WNAE52SJUQCZO5C"
	_, err = ParseMagnetUri(uri)
	if err == nil {
		t.Errorf("Magnet URI with non-BTIH URNs (like \"%v\") are not supported and should return an error", uri)
	}

	// resilience to the broken hash
	uri = "magnet:?xt=urn:btih:this hash is really broken"
	_, err = ParseMagnetUri(uri)
	if err == nil {
		t.Errorf("Failed to detect broken Magnet URI: %v", uri)
	}
}

func TestMagnetize(t *testing.T) {
	t.Parallel()

	mi, err := LoadFromFile("testdata/bootstrap.dat.torrent")
	if err != nil {
		t.Fatal(err)
	}

	info, err := mi.UnmarshalInfo()
	if err != nil {
		t.Fatal(err)
	}
	m := mi.Magnet(nil, &info)

	if m.DisplayName != "bootstrap.dat" {
		t.Errorf("Expected display name to be \"bootstrap.dat\", but got %s", m.DisplayName)
	}

	ih := [20]byte{
		54, 113, 155, 162, 206, 207, 159, 59, 215, 197,
		171, 251, 122, 136, 233, 57, 97, 27, 83, 108,
	}

	if m.InfoHash != ih {
		t.Errorf("Magnet infohash is incorrect")
	}

	trackers := []string{
		"udp://tracker.openbittorrent.com:80",
		"udp://tracker.openbittorrent.com:80",
		"udp://tracker.publicbt.com:80",
		"udp://coppersurfer.tk:6969/announce",
		"udp://open.demonii.com:1337",
		"http://bttracker.crunchbanglinux.org:6969/announce",
	}

	for _, expected := range trackers {
		if !contains(m.Trackers, expected) {
			t.Errorf("Magnet does not contain expected tracker: %s", expected)
		}
	}
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

// Check that we can parse the magnet link generated from a real-world torrent. This was added due
// to a regression in copyParams.
func TestParseSintelMagnet(t *testing.T) {
	t.Parallel()

	mi, err := LoadFromFile("testdata/sintel.torrent")
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	m := mi.Magnet(nil, nil)
	ms := m.String()
	t.Logf("magnet link: %q", ms)
	_, err = ParseMagnetUri(ms)
	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}
	_, err = ParseMagnetV2Uri(ms)
	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}
}
