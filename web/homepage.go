package web

import (
	"fmt"
	"net/http"

	g "maragu.dev/gomponents"
	c "maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
)

func homepage(ntorrents uint) g.Node {
	return c.HTML5(c.HTML5Props{
		Title:       "magnetico",
		Description: "A self-hosted BitTorrent DHT search engine",
		Language:    "en",
		Head: []g.Node{
			Meta(Charset("utf-8")),
			Meta(Name("viewport"), Content("width=device-width, initial-scale=1")),
			Link(Rel("stylesheet"), Href("/static/styles/reset.css")),
			Link(Rel("stylesheet"), Href("/static/styles/essential.css")),
			Link(Rel("stylesheet"), Href("/static/styles/homepage.css")),
		},
		Body: []g.Node{
			Main(
				Div(
					B(g.Text("magnetico")),
					g.Text(" is a self-hosted BitTorrent DHT search engine."),
				),
				Form(
					Action("/torrents"),
					Method("GET"),
					AutoComplete("off"),
					Role("search"),
					Input(
						Type("search"),
						Name("query"),
						Placeholder("Search the BitTorrent DHT"),
						AutoFocus(),
					),
				),
			),
			Footer(
				g.Text(fmt.Sprintf("%d torrents available (see the ", ntorrents)),
				A(Href("/statistics"), g.Text("statistics")),
				g.Text(")"),
			),
		},
	})
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Redirect(w, r, "/", http.StatusMovedPermanently)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	nTorrents, err := database.GetNumberOfTorrents()
	if err != nil {
		http.Error(w, "GetNumberOfTorrents "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err = homepage(nTorrents).Render(w); err != nil {
		http.Error(w, "Homepage render "+err.Error(), http.StatusInternalServerError)
	}
}
