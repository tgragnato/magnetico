package web

import (
	"encoding/hex"
	"html/template"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/tgragnato/magnetico/persistence"
	"golang.org/x/crypto/bcrypt"
)

var (
	// Set a Decoder instance as a package global, because it caches
	// meta-data about structs, and an instance can be shared safely.
	decoder            = schema.NewDecoder()
	templates          map[string]*template.Template
	database           persistence.Database
	credentials        map[string][]byte
	credentialsRWMutex sync.RWMutex // TODO: encapsulate credentials and mutex for safety
)

func makeRouter() *mux.Router {
	apiReadmeHandler, err := NewApiReadmeHandler()
	if err != nil {
		log.Fatalf("Could not initialise readme handler %v", err)
	}
	defer apiReadmeHandler.Close()

	router := mux.NewRouter()
	router.HandleFunc("/",
		BasicAuth(rootHandler))

	router.HandleFunc("/api/v0.1/statistics",
		BasicAuth(apiStatistics))
	router.HandleFunc("/api/v0.1/torrents",
		BasicAuth(apiTorrents))
	router.HandleFunc("/api/v0.1/torrents/{infohash:[a-f0-9]{40}}",
		BasicAuth(apiTorrent))
	router.HandleFunc("/api/v0.1/torrents/{infohash:[a-f0-9]{40}}/filelist",
		BasicAuth(apiFileList))
	router.Handle("/api/v0.1/torrents/{infohash:[a-f0-9]{40}}/readme",
		apiReadmeHandler)

	router.HandleFunc("/feed",
		BasicAuth(feedHandler))
	router.PathPrefix("/static").HandlerFunc(
		BasicAuth(staticHandler))
	router.HandleFunc("/statistics",
		BasicAuth(statisticsHandler))
	router.HandleFunc("/torrents",
		BasicAuth(torrentsHandler))
	router.HandleFunc("/torrents/{infohash:[a-f0-9]{40}}",
		BasicAuth(torrentsInfohashHandler))

	templateFunctions := template.FuncMap{
		"add": func(augend int, addends int) int {
			return augend + addends
		},

		"subtract": func(minuend int, subtrahend int) int {
			return minuend - subtrahend
		},

		"bytesToHex": hex.EncodeToString,

		"unixTimeToYearMonthDay": func(s int64) string {
			tm := time.Unix(s, 0)
			// > Format and Parse use example-based layouts. Usually you’ll use a constant from time
			// > for these layouts, but you can also supply custom layouts. Layouts must use the
			// > reference time Mon Jan 2 15:04:05 MST 2006 to show the pattern with which to
			// > format/parse a given time/string. The example time must be exactly as shown: the
			// > year 2006, 15 for the hour, Monday for the day of the week, etc.
			// https://gobyexample.com/time-formatting-parsing
			// Why you gotta be so weird Go?
			return tm.Format("02/01/2006")
		},

		"humanizeSize": humanize.IBytes,

		"humanizeSizeF": func(s int64) string {
			if s < 0 {
				return ""
			}
			return humanize.IBytes(uint64(s))
		},

		"comma": func(s uint) string {
			return humanize.Comma(int64(s))
		},
	}

	templates = make(map[string]*template.Template)
	templates["feed"] = template.
		Must(template.New("feed").
			Funcs(templateFunctions).
			Parse(string(mustAsset("templates/feed.xml"))))
	templates["homepage"] = template.
		Must(template.New("homepage").
			Funcs(templateFunctions).
			Parse(string(mustAsset("templates/homepage.html"))))

	decoder.IgnoreUnknownKeys(false)
	decoder.ZeroEmpty(true)

	return router
}

// BasicAuth wraps a handler requiring HTTP basic auth for it using the given
// username and password and the specified realm, which shouldn't contain quotes.
//
// Most web browser display a dialog with something like:
//
//	The website says: "<realm>"
//
// Which is really stupid, so you may want to set the realm to a message rather than
// an actual realm.
//
// Source: https://stackoverflow.com/a/39591234/4466589
func BasicAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if len(credentials) == 0 {
			handler(w, r)
			return
		}

		username, password, ok := r.BasicAuth()
		if !ok { // No credentials provided
			authenticate(w)
			return
		}

		credentialsRWMutex.RLock()
		hashedPassword, ok := credentials[username]
		credentialsRWMutex.RUnlock()
		if !ok { // User not found
			authenticate(w)
			return
		}

		if err := bcrypt.CompareHashAndPassword(hashedPassword, []byte(password)); err != nil { // Wrong password
			authenticate(w)
			return
		}

		handler(w, r)
	}
}

func mustAsset(name string) []byte {
	data, err := fs.ReadFile(name)
	if err != nil {
		log.Panicf("Could NOT access the requested resource! THIS IS A BUG, PLEASE REPORT. %v", err)
	}
	return data
}

func authenticate(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="magneticow"`)
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte("Unauthorised.\n"))
}
