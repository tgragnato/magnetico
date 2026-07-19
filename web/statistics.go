package web

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	g "maragu.dev/gomponents"
	c "maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
	"tgragnato.it/magnetico/v2/persistence"
)

func statisticsFrom(n int, unit string) string {
	var dur time.Duration
	switch unit {
	case "hours":
		dur = time.Duration(n) * time.Hour
	case "days":
		dur = time.Duration(n) * 24 * time.Hour
	case "weeks":
		dur = time.Duration(n) * 7 * 24 * time.Hour
	case "months":
		dur = time.Duration(n) * 30 * 24 * time.Hour
	case "years":
		dur = time.Duration(n) * 365 * 24 * time.Hour
	default:
		dur = time.Duration(n) * time.Hour
	}
	from := time.Now().UTC().Add(-dur)
	switch unit {
	case "years":
		return fmt.Sprintf("%d", from.Year())
	case "weeks":
		year, week := from.ISOWeek()
		return fmt.Sprintf("%d-W%02d", year, week)
	case "months":
		return fmt.Sprintf("%d-%02d", from.Year(), int(from.Month()))
	case "days":
		return fmt.Sprintf("%d-%02d-%02d", from.Year(), int(from.Month()), from.Day())
	default: // hours
		return fmt.Sprintf("%d-%02d-%02dT%02d", from.Year(), int(from.Month()), from.Day(), from.Hour())
	}
}

func sortedMapKeys(m map[string]uint64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

type figure struct {
	XMLName    xml.Name   `xml:"figure"`
	Figcaption figcaption `xml:"figcaption"`
	P          *string    `xml:"p,omitempty"`
	SVG        *chartsvg  `xml:"svg,omitempty"`
}

type figcaption struct {
	Strong string `xml:"strong"`
}

type chartsvg struct {
	ViewBox string `xml:"viewBox,attr"`
	Width   string `xml:"width,attr"`
	Style   string `xml:"style,attr"`
	Content string `xml:",innerxml"`
}

func renderChartSVG(title string, data map[string]uint64, yFormatter func(float64) string) string {
	keys := sortedMapKeys(data)

	values := make([]float64, len(keys))
	maxVal := 0.0
	for i, k := range keys {
		v := float64(data[k])
		values[i] = v
		if v > maxVal {
			maxVal = v
		}
	}
	if maxVal == 0 {
		maxVal = 1
	}

	const (
		W      = 800
		H      = 300
		padL   = 80
		padR   = 20
		padT   = 20
		padB   = 50
		chartW = W - padL - padR
		chartH = H - padT - padB
	)

	n := len(keys)
	xStep := 0.0
	if n > 1 {
		xStep = float64(chartW) / float64(n-1)
	}

	var pts strings.Builder
	for i, v := range values {
		x := float64(padL) + float64(i)*xStep
		y := float64(padT+chartH) - (v/maxVal)*float64(chartH)
		if i > 0 {
			pts.WriteByte(' ')
		}
		fmt.Fprintf(&pts, "%.1f,%.1f", x, y)
	}

	var innerSVG strings.Builder

	fmt.Fprintf(&innerSVG, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="currentColor" stroke-width="1"/>`, padL, padT, padL, padT+chartH)
	fmt.Fprintf(&innerSVG, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="currentColor" stroke-width="1"/>`, padL, padT+chartH, W-padR, padT+chartH)

	for i := 0; i <= 4; i++ {
		v := maxVal * float64(i) / 4
		y := float64(padT+chartH) - (v/maxVal)*float64(chartH)

		var escapedLabel strings.Builder
		xml.EscapeText(&escapedLabel, []byte(yFormatter(v)))

		fmt.Fprintf(&innerSVG, `<line x1="%d" y1="%.0f" x2="%d" y2="%.0f" stroke="#ccc" stroke-width="1"/>`, padL, y, W-padR, y)
		fmt.Fprintf(&innerSVG, `<text x="%d" y="%.0f" text-anchor="end" dominant-baseline="middle" font-size="11">%s</text>`, padL-5, y, escapedLabel.String())
	}

	step := 1
	if n > 8 {
		step = (n + 7) / 8
	}
	for i := 0; i < n; i += step {
		x := float64(padL) + float64(i)*xStep
		label := keys[i]
		if len(label) > 13 {
			label = label[:13]
		}

		var escapedLabel strings.Builder
		xml.EscapeText(&escapedLabel, []byte(label))

		fmt.Fprintf(&innerSVG, `<text x="%.0f" y="%d" text-anchor="middle" font-size="10">%s</text>`, x, padT+chartH+16, escapedLabel.String())
	}

	fmt.Fprintf(&innerSVG, `<polyline points="%s" fill="none" stroke="currentColor" stroke-width="2"/>`, pts.String())

	fig := figure{
		Figcaption: figcaption{Strong: title},
		SVG: &chartsvg{
			ViewBox: fmt.Sprintf("0 0 %d %d", W, H),
			Width:   "100%",
			Style:   "display:block;overflow:visible",
			Content: innerSVG.String(),
		},
	}
	if len(keys) == 0 {
		noDataTxt := "No data available"
		fig = figure{
			Figcaption: figcaption{Strong: title},
			P:          &noDataTxt,
			SVG: &chartsvg{
				ViewBox: fmt.Sprintf("0 0 %d %d", W, H),
				Width:   "100%",
				Style:   "display:block;overflow:visible",
				Content: innerSVG.String(),
			},
		}
	}

	output, err := xml.Marshal(fig)
	if err != nil {
		return ""
	}

	return string(output)
}

func statisticsCharts(stats *persistence.Statistics) g.Node {
	nDiscovered := renderChartSVG("Torrents Discovered", stats.NDiscovered, func(v float64) string {
		return fmt.Sprintf("%.0f", v)
	})
	nFiles := renderChartSVG("Files Discovered", stats.NFiles, func(v float64) string {
		return fmt.Sprintf("%.0f", v)
	})
	totalSize := renderChartSVG("Total Size of Files Discovered", stats.TotalSize, func(v float64) string {
		return fmt.Sprintf("%.1f GiB", v/1024/1024/1024)
	})

	return Div(
		ID("charts"),
		Div(Class("graph"), ID("nDiscovered"), g.Raw(nDiscovered)),
		Div(Class("graph"), ID("nFiles"), g.Raw(nFiles)),
		Div(Class("graph"), ID("totalSize"), g.Raw(totalSize)),
	)
}

func statistics(stats *persistence.Statistics) g.Node {
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
			Script(Src("/static/scripts/htmx-2.0.10.js")),
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
					Form(
						g.Attr("hx-get", "/statistics/partial"),
						g.Attr("hx-target", "#charts"),
						g.Attr("hx-trigger", "change"),
						P(
							g.Text("Show statistics for the past ..."),
							Input(
								ID("n"),
								Name("n"),
								Title("maximum number of time units from now backwards"),
								Type("number"),
								Value("24"),
								Min("5"),
								Max("365"),
							),
							Select(
								ID("unit"),
								Name("unit"),
								Title("time unit to be used"),
								Required(),
								Option(Value("hours"), Selected(), g.Text("Hours")),
								Option(Value("days"), g.Text("Days")),
								Option(Value("weeks"), g.Text("Weeks")),
								Option(Value("months"), g.Text("Months")),
								Option(Value("years"), g.Text("Years")),
							),
							g.Text("."),
						),
					),
				),
				statisticsCharts(stats),
			),
		},
	})
}

func statisticsHandler(w http.ResponseWriter, r *http.Request) {
	from := statisticsFrom(24, "hours")
	stats, err := database.GetStatistics(from, 24)
	if err != nil {
		http.Error(w, "GetStatistics "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(ContentType, ContentTypeHtml)
	if err := statistics(stats).Render(w); err != nil {
		http.Error(w, "Statistics render "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func statisticsPartialHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	n := 24
	if q.Has("n") {
		var err error
		n, err = strconv.Atoi(q.Get("n"))
		if err != nil || n <= 0 {
			http.Error(w, "invalid n", http.StatusBadRequest)
			return
		}
	}

	unit := "hours"
	if q.Has("unit") {
		switch q.Get("unit") {
		case "hours", "days", "weeks", "months", "years":
			unit = q.Get("unit")
		default:
			http.Error(w, "invalid unit", http.StatusBadRequest)
			return
		}
	}

	from := statisticsFrom(n, unit)
	stats, err := database.GetStatistics(from, uint(n))
	if err != nil {
		http.Error(w, "GetStatistics "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(ContentType, ContentTypeHtml)
	if err := statisticsCharts(stats).Render(w); err != nil {
		http.Error(w, "render "+err.Error(), http.StatusInternalServerError)
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
