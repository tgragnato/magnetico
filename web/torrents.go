package web

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	g "maragu.dev/gomponents"
	c "maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
	"tgragnato.it/magnetico/v2/persistence"
)

const defaultLimit = 20

func torrentOrderedValue(t persistence.TorrentMetadata, orderBy persistence.OrderingCriteria) float64 {
	switch orderBy {
	case persistence.ByDiscoveredOn:
		return float64(t.DiscoveredOn)
	case persistence.ByRelevance:
		return t.Relevance
	case persistence.ByTotalSize:
		return float64(t.Size)
	case persistence.ByNFiles:
		return float64(t.NFiles)
	default:
		return float64(t.DiscoveredOn)
	}
}

func torrentItem(t persistence.TorrentMetadata) g.Node {
	infohashHex := hex.EncodeToString(t.InfoHash)
	return Li(
		Div(
			H3(A(Href("/torrents/"+infohashHex), g.Text(t.Name))),
			A(
				Href(fmt.Sprintf("magnet:?xt=urn:btih:%s&dn=%s", infohashHex, url.QueryEscape(t.Name))),
				Img(Src("/static/assets/magnet.gif"), Alt("Magnet link")),
				Title("Download this torrent using magnet"),
			),
			Small(g.Text(infohashHex)),
		),
		g.Text(bytesToHuman(t.Size)+", "+time.Unix(t.DiscoveredOn, 0).Format("02/01/2006")),
	)
}

func loadMoreButton(oob bool, query string, epoch int64, orderByStr string, ascending bool, lastID uint64, lastOrderedValue float64, disabled bool) g.Node {
	attrs := []g.Node{ID("load-more")}
	if oob {
		attrs = append(attrs, g.Attr("hx-swap-oob", "true"))
	}
	if disabled {
		return Button(append(attrs, g.Attr("disabled", ""), g.Text("No More Results"))...)
	}
	params := url.Values{}
	params.Set("epoch", strconv.FormatInt(epoch, 10))
	params.Set("lastID", strconv.FormatUint(lastID, 10))
	params.Set("lastOrderedValue", strconv.FormatFloat(lastOrderedValue, 'f', -1, 64))
	params.Set("orderBy", orderByStr)
	params.Set("ascending", strconv.FormatBool(ascending))
	if query != "" {
		params.Set("query", query)
	}
	return Button(append(attrs,
		g.Attr("hx-get", "/torrents/results?"+params.Encode()),
		g.Attr("hx-target", "main ul"),
		g.Attr("hx-swap", "beforeend"),
		g.Text("Load More Results"),
	)...)
}

func torrents(query string, epoch int64, orderByStr string, ascending bool, results []persistence.TorrentMetadata) g.Node {
	title := "Search - magnetico"
	if query != "" {
		title = query + " - magnetico"
	}

	feedHref := "/feed"
	if query != "" {
		feedHref = "/feed?query=" + url.QueryEscape(query)
	}

	items := make([]g.Node, len(results))
	for i, t := range results {
		items[i] = torrentItem(t)
	}

	var button g.Node
	if len(results) < defaultLimit {
		button = loadMoreButton(false, "", 0, "", false, 0, 0, true)
	} else {
		last := results[len(results)-1]
		ob, _ := parseOrderBy(orderByStr)
		button = loadMoreButton(false, query, epoch, orderByStr, ascending, last.ID, torrentOrderedValue(last, ob), false)
	}

	return c.HTML5(c.HTML5Props{
		Title:       title,
		Description: "A self-hosted BitTorrent DHT search engine",
		Language:    "en",
		Head: []g.Node{
			Meta(Charset("utf-8")),
			Meta(Name("viewport"), Content("width=device-width, initial-scale=1")),
			Link(Rel("stylesheet"), Href("/static/styles/reset.css")),
			Link(Rel("stylesheet"), Href("/static/styles/essential.css")),
			Link(Rel("stylesheet"), Href("/static/styles/torrents.css")),
			Script(Src("/static/scripts/htmx-2.0.10.js")),
		},
		Body: []g.Node{
			Header(
				Div(A(Href("/"), B(g.Text("magnetico")))),
				Form(
					Action("/torrents"),
					Method("get"),
					AutoComplete("off"),
					Role("search"),
					Input(
						Type("search"),
						Name("query"),
						Placeholder("Search the BitTorrent DHT"),
						g.If(query != "", Value(query)),
					),
				),
				Div(
					A(
						Href(feedHref),
						ID("feed-anchor"),
						Img(Src("/static/assets/feed.png"), Alt("RSS feed icon"), Title("subscribe to the RSS feed")),
						g.Text("subscribe"),
					),
				),
			),
			Main(Ul(items...)),
			Footer(button),
		},
	})
}

func torrentsHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	epoch := time.Now().Unix()

	orderByStr := "DISCOVERED_ON"
	ascending := false
	if query != "" {
		orderByStr = "RELEVANCE"
		ascending = true
	}

	orderBy, _ := parseOrderBy(orderByStr)
	results, err := database.QueryTorrents(query, epoch, orderBy, ascending, defaultLimit, nil, nil)
	if err != nil {
		http.Error(w, "QueryTorrents: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(ContentType, ContentTypeHtml)
	if err := torrents(query, epoch, orderByStr, ascending, results).Render(w); err != nil {
		http.Error(w, "Torrents render "+err.Error(), http.StatusInternalServerError)
	}
}

func torrentsResultsHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	query := q.Get("query")

	epoch := time.Now().Unix()
	if q.Has("epoch") {
		var err error
		epoch, err = strconv.ParseInt(q.Get("epoch"), 10, 64)
		if err != nil {
			http.Error(w, "invalid epoch: "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	orderByStr := "DISCOVERED_ON"
	if query != "" {
		orderByStr = "RELEVANCE"
	}
	if q.Has("orderBy") {
		orderByStr = q.Get("orderBy")
	}
	orderBy, err := parseOrderBy(orderByStr)
	if err != nil {
		http.Error(w, "invalid orderBy: "+err.Error(), http.StatusBadRequest)
		return
	}

	ascending := false
	if q.Has("ascending") {
		ascending, err = strconv.ParseBool(q.Get("ascending"))
		if err != nil {
			http.Error(w, "invalid ascending: "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	var lastOrderedValue *float64
	var lastID *uint64
	if q.Has("lastID") && q.Has("lastOrderedValue") {
		lov, err := strconv.ParseFloat(q.Get("lastOrderedValue"), 64)
		if err != nil {
			http.Error(w, "invalid lastOrderedValue: "+err.Error(), http.StatusBadRequest)
			return
		}
		lid, err := strconv.ParseUint(q.Get("lastID"), 10, 64)
		if err != nil {
			http.Error(w, "invalid lastID: "+err.Error(), http.StatusBadRequest)
			return
		}
		lastOrderedValue = &lov
		lastID = &lid
	}

	results, err := database.QueryTorrents(query, epoch, orderBy, ascending, defaultLimit, lastOrderedValue, lastID)
	if err != nil {
		http.Error(w, "QueryTorrents: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var sb strings.Builder
	for _, t := range results {
		if err := torrentItem(t).Render(&sb); err != nil {
			http.Error(w, "render: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	var button g.Node
	if len(results) < defaultLimit {
		button = loadMoreButton(true, "", 0, "", false, 0, 0, true)
	} else {
		last := results[len(results)-1]
		button = loadMoreButton(true, query, epoch, orderByStr, ascending, last.ID, torrentOrderedValue(last, orderBy), false)
	}
	if err := button.Render(&sb); err != nil {
		http.Error(w, "render: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(ContentType, ContentTypeHtml)
	fmt.Fprint(w, sb.String())
}

func apiTorrents(w http.ResponseWriter, r *http.Request) {
	// @lastOrderedValue AND @lastID are either both supplied or neither of them should be supplied
	// at all; and if that is NOT the case, then return an error.
	if q := r.URL.Query(); q.Get("lastOrderedValue") == "" && q.Get("lastID") != "" || q.Get("lastOrderedValue") != "" && q.Get("lastID") == "" {
		http.Error(w, "`lastOrderedValue`, `lastID` must be supplied altogether, if supplied.", http.StatusBadRequest)
		return
	}

	var tq struct {
		Epoch            int64    `schema:"epoch"`
		Query            string   `schema:"query"`
		OrderBy          string   `schema:"orderBy"`
		Ascending        bool     `schema:"ascending"`
		LastOrderedValue *float64 `schema:"lastOrderedValue"`
		LastID           *uint64  `schema:"lastID"`
		Limit            uint64   `schema:"limit"`
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "error while parsing the URL: "+err.Error(), http.StatusBadRequest)
		return
	}

	if r.Form.Has("epoch") {
		tq.Epoch, err = strconv.ParseInt(r.Form.Get("epoch"), 10, 64)
		if err != nil {
			http.Error(w, "error while parsing the URL: "+err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		tq.Epoch = time.Now().Unix()
	}

	tq.Query = r.Form.Get("query")
	tq.OrderBy = r.Form.Get("orderBy")

	var orderBy persistence.OrderingCriteria
	if tq.OrderBy == "" {
		if tq.Query == "" {
			orderBy = persistence.ByDiscoveredOn
		} else {
			orderBy = persistence.ByRelevance
		}
	} else {
		var err error
		orderBy, err = parseOrderBy(tq.OrderBy)
		if err != nil {
			http.Error(w, "error while parsing the URL: "+err.Error(), http.StatusBadRequest)
		}
	}

	if r.Form.Has("ascending") {
		tq.Ascending, err = strconv.ParseBool(r.Form.Get("ascending"))
		if err != nil {
			http.Error(w, "error while parsing the URL: "+err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		tq.Ascending = true
	}

	if r.Form.Has("lastOrderedValue") {
		lastOrderedValue, err := strconv.ParseFloat(r.Form.Get("lastOrderedValue"), 64)
		if err != nil {
			http.Error(w, "error while parsing the URL: "+err.Error(), http.StatusBadRequest)
			return
		}
		tq.LastOrderedValue = &lastOrderedValue
	}

	if r.Form.Has("lastID") {
		lastID, err := strconv.ParseUint(r.Form.Get("lastID"), 10, 64)
		if err != nil {
			http.Error(w, "error while parsing the URL: "+err.Error(), http.StatusBadRequest)
			return
		}
		tq.LastID = &lastID
	}

	if r.Form.Has("limit") {
		tq.Limit, err = strconv.ParseUint(r.Form.Get("limit"), 10, 64)
		if err != nil {
			http.Error(w, "error while parsing the URL: "+err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		tq.Limit = 20
	}

	torrents, err := database.QueryTorrents(
		tq.Query, tq.Epoch, orderBy,
		tq.Ascending, tq.Limit, tq.LastOrderedValue, tq.LastID)
	if err != nil {
		http.Error(w, "QueryTorrents: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(ContentType, ContentTypeJson)
	if err = json.NewEncoder(w).Encode(torrents); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func apiTorrentsTotal(w http.ResponseWriter, r *http.Request) {
	// @lastOrderedValue AND @lastID are either both supplied or neither of them should be supplied
	// at all; and if that is NOT the case, then return an error.
	if q := r.URL.Query(); q.Get("lastOrderedValue") == "" && q.Get("lastID") != "" || q.Get("lastOrderedValue") != "" && q.Get("lastID") == "" {
		http.Error(w, "`lastOrderedValue`, `lastID` must be supplied altogether, if supplied.", http.StatusBadRequest)
		return
	}

	var tq struct {
		Epoch int64  `schema:"epoch"`
		Query string `schema:"query"`
		// Controls compatibility. If this parameter is not provided or is set to false, the old logic is executed.
		// If set to true, the new logic is enabled.
		// The old logic returns a single number, while the new logic returns a map[string]any JSON object.
		NewLogic bool `schema:"newLogic"`
		// Due to potential ambiguity in the function name apiTorrentsTotal, the QueryType parameter was introduced.
		// To use this parameter, `NewLogic=true` is required. This parameter specifies the type of query we are performing.
		// For example, `byAll` indicates querying the total count from the database,
		// while `byKeyword` indicates querying the total count that matches the given query.
		QueryType string `schema:"queryType"`
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "error while parsing the URL: "+err.Error(), http.StatusBadRequest)
		return
	}

	if r.Form.Has("epoch") {
		tq.Epoch, err = strconv.ParseInt(r.Form.Get("epoch"), 10, 64)
		if err != nil {
			http.Error(w, "error while parsing the URL: "+err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		http.Error(w, "lack required parameters while parsing the URL: `epoch`", http.StatusBadRequest)
		return
	}

	if r.Form.Has("newLogic") {
		tq.NewLogic, err = strconv.ParseBool(r.Form.Get("newLogic"))
		if err != nil {
			http.Error(w, "error while parsing the URL: "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	tq.Query = r.Form.Get("query")
	tq.QueryType = r.Form.Get("queryType")

	w.Header().Set(ContentType, ContentTypeJson)

	if !tq.NewLogic {

		torrentsTotal, err := database.GetNumberOfQueryTorrents(tq.Query, tq.Epoch)
		if err != nil {
			http.Error(w, "GetNumberOfQueryTorrents: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if err = json.NewEncoder(w).Encode(torrentsTotal); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	queryCountType, err := parseQueryCountType(tq.QueryType)
	if err != nil {
		http.Error(w, "error while parsing the URL: "+err.Error(), http.StatusBadRequest)
		return
	}

	var results map[string]any

	switch queryCountType {
	case CountQueryTorrentsByKeyword:
		total, err := database.GetNumberOfQueryTorrents(tq.Query, tq.Epoch)
		if err != nil {
			http.Error(w, "GetNumberOfQueryTorrents: "+err.Error(), http.StatusInternalServerError)
			return
		}
		results = map[string]any{"queryType": "byKeyword", "data": total}

	default:
		http.Error(w, "no suitable queryType query was matched", http.StatusBadRequest)
		return
	}

	if err = json.NewEncoder(w).Encode(results); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func parseOrderBy(s string) (persistence.OrderingCriteria, error) {
	switch s {
	case "RELEVANCE":
		return persistence.ByRelevance, nil

	case "TOTAL_SIZE":
		return persistence.ByTotalSize, nil

	case "DISCOVERED_ON":
		return persistence.ByDiscoveredOn, nil

	case "N_FILES":
		return persistence.ByNFiles, nil

	case "UPDATED_ON":
		return persistence.ByUpdatedOn, nil

	case "N_SEEDERS":
		return persistence.ByNSeeders, nil

	case "N_LEECHERS":
		return persistence.ByNLeechers, nil

	default:
		return persistence.ByDiscoveredOn, fmt.Errorf("unknown orderBy string: %s", s)
	}
}

type CountQueryTorrentsType uint8

const (
	CountQueryTorrentsByAll CountQueryTorrentsType = iota
	CountQueryTorrentsByKeyword
)

func parseQueryCountType(s string) (CountQueryTorrentsType, error) {
	switch s {
	case "byKeyword":
		return CountQueryTorrentsByKeyword, nil
	case "byAll":
		return CountQueryTorrentsByAll, nil
	default:
		return CountQueryTorrentsByKeyword, fmt.Errorf("unknown queryType string: %s", s)
	}
}
