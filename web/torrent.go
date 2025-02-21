package web

import (
	"encoding/json"
	"net/http"

	g "maragu.dev/gomponents"
	c "maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
)

func torrent() g.Node {
	return c.HTML5(c.HTML5Props{
		Title:       "Loading ... - magnetico",
		Description: "A self-hosted BitTorrent DHT search engine",
		Language:    "en",
		Head: []g.Node{
			Meta(Charset("utf-8")),
			Meta(Name("viewport"), Content("width=device-width, initial-scale=1")),
			Link(Rel("stylesheet"), Href("/static/styles/vanillatree-v0.0.3.css")),
			Link(Rel("stylesheet"), Href("/static/styles/reset.css")),
			Link(Rel("stylesheet"), Href("/static/styles/essential.css")),
			Link(Rel("stylesheet"), Href("/static/styles/torrent.css")),
			Script(Src("/static/scripts/naturalSort-v0.8.1.js")),
			Script(Src("/static/scripts/mustache-v2.3.0.min.js")),
			Script(Src("/static/scripts/vanillatree-v0.0.3.js")),
			Script(Defer(), Src("/static/scripts/common.js")),
			Script(Defer(), Src("/static/scripts/torrent.js")),
			Script(
				ID("main-template"),
				Type("text/x-handlebars-template"),
				Div(
					ID("title"),
					H2(g.Text("{{ name }}")),
					A(
						Href("magnet:?xt=urn:btih:{{ infoHash }}&dn={{ name }}"),
						Img(
							Src("/static/assets/magnet.gif"),
							Alt("Magnet link"),
							Title("Download this torrent using magnet"),
						),
						Small(g.Text("{{ infoHash }}")),
					),
				),
				Table(
					Tr(
						Th(
							g.Attr(("scope"), "row"),
							g.Text("Size"),
						),
						Td(g.Text("{{ sizeHumanised }}")),
					),
					Tr(
						Th(
							g.Attr(("scope"), "row"),
							g.Text("Discovered on"),
						),
						Td(g.Text("{{ discoveredOn }}")),
					),
					Tr(
						Th(
							g.Attr(("scope"), "row"),
							g.Text("Files"),
						),
						Td(g.Text("{{ nFiles }}")),
					),
				),
				H3(g.Text("Files")),
				Div(ID("fileTree")),
			),
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
			Main(),
		},
	})
}

func torrentsInfohashHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(ContentType, ContentTypeHtml)
	if err := torrent().Render(w); err != nil {
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
