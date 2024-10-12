package metainfo

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"

	"github.com/multiformats/go-multihash"

	"tgragnato.it/magnetico/types/infohash"
	infohash_v2 "tgragnato.it/magnetico/types/infohash-v2"
)

// Magnet link components.
type MagnetV2 struct {
	InfoHash    infohash.T // Expected in this implementation
	V2InfoHash  infohash_v2.T
	Trackers    []string   // "tr" values
	DisplayName string     // "dn" value, if not empty
	Params      url.Values // All other values, such as "x.pe", "as", "xs" etc.
}

const (
	btmhPrefix = "urn:btmh:"
)

func (m MagnetV2) String() string {
	// Deep-copy m.Params
	vs := make(url.Values, len(m.Params)+len(m.Trackers)+2)
	for k, v := range m.Params {
		vs[k] = append([]string(nil), v...)
	}

	for _, tr := range m.Trackers {
		vs.Add("tr", tr)
	}
	if m.DisplayName != "" {
		vs.Add("dn", m.DisplayName)
	}

	// Transmission and Deluge both expect "urn:btih:" to be unescaped. Deluge wants it to be at the
	// start of the magnet link. The InfoHash field is expected to be BitTorrent in this
	// implementation.
	u := url.URL{
		Scheme: "magnet",
	}
	var queryParts []string

	if !reflect.DeepEqual(m.InfoHash, infohash.T{}) {
		queryParts = append(queryParts, "xt="+btihPrefix+m.InfoHash.HexString())
	}
	if !reflect.DeepEqual(m.V2InfoHash, infohash_v2.T{}) {
		queryParts = append(
			queryParts,
			"xt="+btmhPrefix+infohash_v2.ToMultihash(m.V2InfoHash).HexString(),
		)
	}
	if rem := vs.Encode(); rem != "" {
		queryParts = append(queryParts, rem)
	}
	u.RawQuery = strings.Join(queryParts, "&")
	return u.String()
}

// ParseMagnetUri parses Magnet-formatted URIs into a Magnet instance
func ParseMagnetV2Uri(uri string) (m MagnetV2, err error) {
	u, err := url.Parse(uri)
	if err != nil {
		err = fmt.Errorf("error parsing uri: %w", err)
		return
	}
	if u.Scheme != "magnet" {
		err = fmt.Errorf("unexpected scheme %q", u.Scheme)
		return
	}
	q := u.Query()
	for _, xt := range q["xt"] {
		if hashStr, found := strings.CutPrefix(xt, btihPrefix); found {
			if !reflect.DeepEqual(m.InfoHash, infohash.T{}) {
				err = errors.New("more than one infohash found in magnet link")
				return
			}
			m.InfoHash, err = parseEncodedV1Infohash(hashStr)
			if err != nil {
				err = fmt.Errorf("error parsing infohash %q: %w", hashStr, err)
				return
			}
		} else if hashStr, found := strings.CutPrefix(xt, btmhPrefix); found {
			if !reflect.DeepEqual(m.V2InfoHash, infohash_v2.T{}) {
				err = errors.New("more than one infohash found in magnet link")
				return
			}
			m.V2InfoHash, err = parseV2Infohash(hashStr)
			if err != nil {
				err = fmt.Errorf("error parsing infohash %q: %w", hashStr, err)
				return
			}
		} else {
			lazyAddParam(&m.Params, "xt", xt)
		}
	}
	q.Del("xt")
	m.DisplayName = popFirstValue(q, "dn")
	m.Trackers = q["tr"]
	q.Del("tr")
	// Add everything we haven't consumed.
	copyParams(&m.Params, q)
	return
}

func lazyAddParam(vs *url.Values, k, v string) {
	if *vs == nil {
		*vs = make(url.Values)
	}
	vs.Add(k, v)
}

func copyParams(dest *url.Values, src url.Values) {
	for k, vs := range src {
		for _, v := range vs {
			lazyAddParam(dest, k, v)
		}
	}
}

func parseV2Infohash(encoded string) (ih infohash_v2.T, err error) {
	b, err := hex.DecodeString(encoded)
	if err != nil {
		return
	}
	mh, err := multihash.Decode(b)
	if err != nil {
		return
	}
	if mh.Code != multihash.SHA2_256 || mh.Length != infohash_v2.Size || len(mh.Digest) != infohash_v2.Size {
		err = errors.New("bad multihash")
		return
	}
	n := copy(ih[:], mh.Digest)
	if n != infohash_v2.Size {
		return infohash_v2.T{}, errors.New("decoded xt length != 32")
	}
	return
}
