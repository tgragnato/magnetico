package web

import (
	"encoding/json"
	"net/http"
	"strconv"

	g "maragu.dev/gomponents"
	c "maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
)

func statistics() g.Node {
	return c.HTML5(c.HTML5Props{
		Title:       "Statistics - magnetico",
		Description: "A self-hosted BitTorrent DHT search engine",
		Language:    "en",
		Head: []g.Node{
			Meta(Charset("utf-8")),
			Meta(Name("viewport"), Content("width=device-width, initial-scale=1")),
			Link(Rel("stylesheet"), Href("/static/styles/reset.css")),
			Link(Rel("stylesheet"), Href("/static/styles/essential.css")),
			Link(Rel("stylesheet"), Href("/static/styles/statistics.css")),
			Script(Defer(), Src("/static/scripts/plotly-v1.26.1.min.js")),
			Script(Defer(), Src("/static/scripts/common.js")),
			Script(Defer(), Src("/static/scripts/statistics.js")),
		},
		Body: []g.Node{
			Header(
				Div(
					A(
						Href("/"),
						B(g.Text("magnetico")),
					),
				),
			),
			Main(
				Div(
					ID("options"),
					P(
						g.Text("Show statistics for the past ..."),
						Input(
							ID("n"),
							Title("maximum number of time units from now backwards"),
							Type("number"),
							Value("24"),
							Min("5"),
							Max("365"),
						),
						Select(
							ID("unit"),
							Title("time unit to be used"),
							Required(),
							Option(
								Value("hours"),
								Selected(),
								g.Text("Hours"),
							),
							Option(
								Value("days"),
								g.Text("Days"),
							),
							Option(
								Value("weeks"),
								g.Text("Weeks"),
							),
							Option(
								Value("months"),
								g.Text("Months"),
							),
							Option(
								Value("years"),
								g.Text("Years"),
							),
							g.Text("."),
						),
					),
				),
				Div(
					Class("graph"),
					ID("nDiscovered"),
				),
				Div(
					Class("graph"),
					ID("nFiles"),
				),
				Div(
					Class("graph"),
					ID("totalSize"),
				),
			),
		},
	})
}

func statisticsHandler(w http.ResponseWriter, r *http.Request) {
	err := statistics().Render(w)
	if err != nil {
		http.Error(w, "Statistics render "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func apiStatistics(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")

	var n int64
	nStr := r.URL.Query().Get("n")
	if nStr == "" {
		n = 0
	} else {
		var err error
		n, err = strconv.ParseInt(nStr, 10, 32)
		if err != nil {
			http.Error(w, "Couldn't parse n: "+err.Error(), http.StatusBadRequest)
			return
		} else if n <= 0 {
			http.Error(w, "n must be a positive number", http.StatusBadRequest)
			return
		}
	}

	stats, err := database.GetStatistics(from, uint(n))
	if err != nil {
		http.Error(w, "GetStatistics "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(ContentType, ContentTypeJson)
	if err = json.NewEncoder(w).Encode(stats); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
