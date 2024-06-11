package web

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/storage"
	"github.com/gorilla/mux"
	"github.com/tgragnato/magnetico/persistence"
	"golang.org/x/text/encoding/charmap"
)

const (
	RequestLimitSize = 50 * 1024
	MsgJsonError     = "JSON encode error"
	MsgCantDecode    = "Couldn't decode infohash"
	NfoSuffix        = ".nfo"
	MagnetPrefix     = "magnet:?xt=urn:btih:"
)

type ApiReadmeHandler struct {
	client  *torrent.Client
	tempDir string
}

func NewApiReadmeHandler() (*ApiReadmeHandler, error) {
	h := new(ApiReadmeHandler)
	var err error

	h.tempDir, err = os.MkdirTemp("", "magneticod_")
	if err != nil {
		return nil, err
	}

	config := torrent.NewDefaultClientConfig()
	config.ListenPort = 0
	config.DefaultStorage = storage.NewFileByInfoHash(h.tempDir)

	h.client, err = torrent.NewClient(config)
	if err != nil {
		_ = os.RemoveAll(h.tempDir)
		return nil, err
	}

	return h, nil
}

func (h *ApiReadmeHandler) Close() {
	h.client.Close()
	_ = os.RemoveAll(h.tempDir)
}

func (h *ApiReadmeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	infohashHex := mux.Vars(r)["infohash"]

	infohash, err := hex.DecodeString(infohashHex)
	if err != nil {
		respondError(w, http.StatusBadRequest, "%s: %s", MsgCantDecode, err.Error())
		return
	}

	files, err := database.GetFiles(infohash)
	if err != nil {
		log.Printf("GetFiles error %v", err)
		respondError(w, http.StatusInternalServerError, "Internal Server Error")
	}

	ok := false
	for _, file := range files {
		if strings.HasSuffix(file.Path, NfoSuffix) {
			ok = true
			break
		}
		if strings.Contains(file.Path, "read") {
			ok = true
			break
		}
	}

	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	log.Println("README")

	t, err := h.client.AddMagnet(MagnetPrefix + infohashHex)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer t.Drop()

	log.Println("WAITING FOR INFO")

	select {
	case <-t.GotInfo():

	case <-time.After(30 * time.Second):
		respondError(w, http.StatusInternalServerError, "Timeout")
		return
	}

	log.Println("GOT INFO!")

	t.CancelPieces(0, t.NumPieces())

	var file *torrent.File
	for _, file = range t.Files() {
		filePath := file.Path()
		if strings.HasSuffix(filePath, NfoSuffix) {
			break
		} else if strings.Contains(filePath, "read") {
			break
		}
	}

	if file == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Cancel if the file is larger than limit
	if file.Length() > RequestLimitSize {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}

	file.Download()

	reader := file.NewReader()
	content := make([]byte, file.Length())
	_, err = io.ReadFull(reader, content)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	if strings.HasSuffix(file.Path(), NfoSuffix) {
		content, err = charmap.CodePage437.NewDecoder().Bytes(content)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	// Because .nfo files are right padded with \x00'es.
	content = bytes.TrimRight(content, "\x00")

	w.Header().Set(ContentType, ContentTypeText)
	_, _ = w.Write(content)
}

func apiTorrents(w http.ResponseWriter, r *http.Request) {
	// @lastOrderedValue AND @lastID are either both supplied or neither of them should be supplied
	// at all; and if that is NOT the case, then return an error.
	if q := r.URL.Query(); !((q.Get("lastOrderedValue") != "" && q.Get("lastID") != "") ||
		(q.Get("lastOrderedValue") == "" && q.Get("lastID") == "")) {
		respondError(w, http.StatusBadRequest, "`lastOrderedValue`, `lastID` must be supplied altogether, if supplied.")
		return
	}

	var tq struct {
		Epoch            *int64   `schema:"epoch"`
		Query            *string  `schema:"query"`
		OrderBy          *string  `schema:"orderBy"`
		Ascending        *bool    `schema:"ascending"`
		LastOrderedValue *float64 `schema:"lastOrderedValue"`
		LastID           *uint64  `schema:"lastID"`
		Limit            *uint    `schema:"limit"`
	}
	if err := decoder.Decode(&tq, r.URL.Query()); err != nil {
		respondError(w, http.StatusBadRequest, "error while parsing the URL: %s", err.Error())
		return
	}

	if tq.Query == nil {
		tq.Query = new(string)
		*tq.Query = ""
	}

	if tq.Epoch == nil {
		tq.Epoch = new(int64)
		*tq.Epoch = time.Now().Unix() // epoch, if not supplied, is NOW.
	} else if *tq.Epoch <= 0 {
		respondError(w, http.StatusBadRequest, "epoch must be greater than 0")
		return
	}

	if tq.Ascending == nil {
		tq.Ascending = new(bool)
		*tq.Ascending = true
	}

	var orderBy persistence.OrderingCriteria
	if tq.OrderBy == nil {
		if *tq.Query == "" {
			orderBy = persistence.ByDiscoveredOn
		} else {
			orderBy = persistence.ByRelevance
		}
	} else {
		var err error
		orderBy, err = parseOrderBy(*tq.OrderBy)
		if err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	if tq.Limit == nil {
		tq.Limit = new(uint)
		*tq.Limit = 20
	}

	torrents, err := database.QueryTorrents(
		*tq.Query, *tq.Epoch, orderBy,
		*tq.Ascending, *tq.Limit, tq.LastOrderedValue, tq.LastID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "query error: %s", err.Error())
		return
	}

	w.Header().Set(ContentType, ContentTypeJson)
	if err = json.NewEncoder(w).Encode(torrents); err != nil {
		log.Printf("%s %v", MsgJsonError, err)
	}
}

func apiTorrent(w http.ResponseWriter, r *http.Request) {
	infohashHex := mux.Vars(r)["infohash"]

	infohash, err := hex.DecodeString(infohashHex)
	if err != nil {
		respondError(w, http.StatusBadRequest, "%s: %s", MsgCantDecode, err.Error())
		return
	}

	torrentMetadata, err := database.GetTorrent(infohash)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "couldn't get torrent: %s", err.Error())
		return
	} else if torrentMetadata == nil {
		respondError(w, http.StatusNotFound, "Not found")
		return
	}

	w.Header().Set(ContentType, ContentTypeJson)
	if err = json.NewEncoder(w).Encode(torrentMetadata); err != nil {
		log.Printf("%s %v", MsgJsonError, err)
	}
}

func apiFileList(w http.ResponseWriter, r *http.Request) {
	infohashHex := mux.Vars(r)["infohash"]

	infohash, err := hex.DecodeString(infohashHex)
	if err != nil {
		respondError(w, http.StatusBadRequest, "%s: %s", MsgCantDecode, err.Error())
		return
	}

	files, err := database.GetFiles(infohash)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "couldn't get files: %s", err.Error())
		return
	} else if files == nil {
		respondError(w, http.StatusNotFound, "not found")
		return
	}

	w.Header().Set(ContentType, ContentTypeJson)
	if err = json.NewEncoder(w).Encode(files); err != nil {
		log.Printf("%s %v", MsgJsonError, err)
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
			respondError(w, http.StatusBadRequest, "couldn't parse n: %s", err.Error())
			return
		} else if n <= 0 {
			respondError(w, http.StatusBadRequest, "n must be a positive number")
			return
		}
	}

	stats, err := database.GetStatistics(from, uint(n))
	if err != nil {
		respondError(w, http.StatusBadRequest, "error while getting statistics: %s", err.Error())
		return
	}

	w.Header().Set(ContentType, ContentTypeJson)
	if err = json.NewEncoder(w).Encode(stats); err != nil {
		log.Printf("%s %v", MsgJsonError, err)
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
