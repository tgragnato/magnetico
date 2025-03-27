package web

import (
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"tgragnato.it/magnetico/v2/persistence"
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

	count := uint64(20)
	if len(r.URL.Query()["count"]) == 1 {
		i, err := strconv.Atoi(r.URL.Query()["count"][0])
		if err == nil && i > 0 {
			count = uint64(i)
		}
	}

	if query == "" {
		title = "Most recent torrents - magnetico"
	} else {
		title = query + " - magnetico"
	}

	torrents, err := database.QueryTorrents(
		query,
		time.Now().Unix(),
		persistence.ByDiscoveredOn,
		false,
		count,
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
					"magnet:?xt=urn:btih:%s&dn=%s",
					infohash,
					torrent.Name,
				),
				Type: "application/x-bittorrent",
			},
		})
	}

	output, err := xml.Marshal(feed)
	if err != nil {
		http.Error(w, "enconding XML "+err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = w.Write([]byte("<?xml version=\"1.0\" encoding=\"utf-8\" standalone=\"yes\"?>\n"))
	if err != nil {
		http.Error(w, "writing XML "+err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = w.Write(output)
	if err != nil {
		http.Error(w, "writing XML "+err.Error(), http.StatusInternalServerError)
		return
	}
}
