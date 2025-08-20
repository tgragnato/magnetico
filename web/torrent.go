package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	g "maragu.dev/gomponents"
	c "maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
	"tgragnato.it/magnetico/v2/persistence"
)

func torrent(infohash string, torrentMetadata persistence.TorrentMetadata, files []persistence.File) g.Node {
	return c.HTML5(c.HTML5Props{
		Title:       infohash + " - magnetico",
		Description: "A self-hosted BitTorrent DHT search engine",
		Language:    "en",
		Head: []g.Node{
			Meta(Charset("utf-8")),
			Meta(Name("viewport"), Content("width=device-width, initial-scale=1")),
			Link(Rel("stylesheet"), Href("/static/styles/reset.css")),
			Link(Rel("stylesheet"), Href("/static/styles/essential.css")),
			Link(Rel("stylesheet"), Href("/static/styles/torrent.css")),
		},
		Body: []g.Node{
			Header(
				Div(
					A(
						Href("/"),
						B(g.Text("magnetico")),
					),
				),
				Form(
					Action("/torrents"),
					Method("get"),
					AutoComplete("off"),
					Role("search"),
					Input(
						Type("search"),
						Name("query"),
						Placeholder("Search the BitTorrent DHT"),
					),
				),
			),
			Main(
				Div(
					Div(
						ID("title"),
						H2(g.Text(fmt.Sprintf("%v", torrentMetadata.Name))),
						A(
							Href(fmt.Sprintf("magnet:?xt=urn:btih:%x&dn=%v", infohash, torrentMetadata.Name)),
							Img(
								Src("/static/assets/magnet.gif"),
								Alt("Magnet link"),
								Title("Download this torrent using magnet"),
							),
							Small(g.Text(fmt.Sprintf("%x", infohash))),
						),
					),
					Table(
						Tr(
							Th(g.Attr(("scope"), "row"), g.Text("Size")),
							Td(g.Text(bytesToHuman(torrentMetadata.Size))),
						),
						Tr(
							Th(g.Attr(("scope"), "row"), g.Text("Discovered on")),
							Td(g.Text(fmt.Sprintf("%v", time.Unix(torrentMetadata.DiscoveredOn, 0).Format("2006-01-02")))),
						),
						Tr(
							Th(g.Attr(("scope"), "row"), g.Text("Files")),
							Td(g.Text(fmt.Sprintf("%v", torrentMetadata.NFiles))),
						),
					),
				),
				Div(
					H3(g.Text("Files")),
					Div(
						Ul(
							g.Map(files, func(file persistence.File) g.Node {
								return Li(
									Span(
										g.Text(file.Path + " (" + bytesToHuman(uint64(file.Size)) + ")"),
									),
								)
							}),
						),
					),
				),
			),
		},
	})
}

func torrentsInfohashHandler(w http.ResponseWriter, r *http.Request) {
	var infohash []byte
	val := r.Context().Value(InfohashKey)
	if val == nil {
		infohash = []byte("")
	} else {
		infohash = val.([]byte)
	}

	torrentMetadata, err := database.GetTorrent(infohash)
	if err != nil {
		http.Error(w, "Couldn't get torrent: "+err.Error(), http.StatusInternalServerError)
		return
	} else if torrentMetadata == nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	files, err := database.GetFiles(infohash)
	if err != nil {
		http.Error(w, "Couldn't get files: "+err.Error(), http.StatusInternalServerError)
		return
	} else if files == nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	w.Header().Set(ContentType, ContentTypeHtml)
	if err := torrent(string(infohash), *torrentMetadata, files).Render(w); err != nil {
		http.Error(w, "Torrent render "+err.Error(), http.StatusInternalServerError)
	}
}

func apiTorrent(w http.ResponseWriter, r *http.Request) {
	infohash := r.Context().Value(InfohashKey).([]byte)

	torrentMetadata, err := database.GetTorrent(infohash)
	if err != nil {
		http.Error(w, "GetTorrent "+err.Error(), http.StatusInternalServerError)
		return
	} else if torrentMetadata == nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	w.Header().Set(ContentType, ContentTypeJson)
	if err = json.NewEncoder(w).Encode(torrentMetadata); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func apiFileList(w http.ResponseWriter, r *http.Request) {
	infohash := r.Context().Value(InfohashKey).([]byte)

	files, err := database.GetFiles(infohash)
	if err != nil {
		http.Error(w, "Couldn't get files: "+err.Error(), http.StatusInternalServerError)
		return
	} else if files == nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	w.Header().Set(ContentType, ContentTypeJson)
	if err = json.NewEncoder(w).Encode(files); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
