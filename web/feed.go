package web

import (
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"net/http"
	"time"

	"github.com/tgragnato/magnetico/persistence"
)

type Feed struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel Item
}

type Item struct {
	Title    string    `xml:"item>title"`
	Torrents []Torrent `xml:"item>item"`
}

type Torrent struct {
	Title     string `xml:"title"`
	GUID      string `xml:"guid"`
	Enclosure struct {
		URL  string `xml:"url,attr"`
		Type string `xml:"type,attr"`
	} `xml:"enclosure"`
}

func feedHandler(w http.ResponseWriter, r *http.Request) {
	var query, title string
	switch len(r.URL.Query()["query"]) {
	case 0:
		query = ""
	case 1:
		query = r.URL.Query()["query"][0]
	default:
		http.Error(w, "query supplied multiple times!", http.StatusBadRequest)
		return
	}

	if query == "" {
		title = "Most recent torrents - magnetico"
	} else {
		title = "`" + query + "` - magnetico"
	}

	torrents, err := database.QueryTorrents(
		query,
		time.Now().Unix(),
		persistence.ByDiscoveredOn,
		true,
		20,
		nil,
		nil,
	)
	if err != nil {
		http.Error(w, "query torrent "+err.Error(), http.StatusInternalServerError)
		return
	}

	feed := Feed{
		Version: "2.0",
		Channel: Item{
			Title:    title,
			Torrents: []Torrent{},
		},
	}

	for _, torrent := range torrents {
		infohash := hex.EncodeToString(torrent.InfoHash)
		feed.Channel.Torrents = append(feed.Channel.Torrents, Torrent{
			Title: torrent.Name,
			GUID:  infohash,
			Enclosure: struct {
				URL  string `xml:"url,attr"`
				Type string `xml:"type,attr"`
			}{
				URL: fmt.Sprintf(
					"magnet:?xt=urn:btih:%s&amp;dn=%s",
					infohash,
					torrent.Name,
				),
				Type: "application/x-bittorrent",
			},
		})
	}

	output, err := xml.MarshalIndent(feed, "", "  ")
	if err != nil {
		http.Error(w, "enconding XML "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("<?xml version=\"1.0\" encoding=\"utf-8\" standalone=\"yes\"?>\n"))
	w.Write(output)
}
