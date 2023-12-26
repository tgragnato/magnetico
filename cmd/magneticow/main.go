package main

import (
	"bufio"
	"bytes"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/jessevdk/go-flags"
	"golang.org/x/crypto/bcrypt"

	"github.com/tgragnato/magnetico/persistence"
)

//go:embed static/** templates/*
var fs embed.FS

var (
	// Set a Decoder instance as a package global, because it caches
	// meta-data about structs, and an instance can be shared safely.
	decoder   = schema.NewDecoder()
	templates map[string]*template.Template
	database  persistence.Database
)

var opts struct {
	Addr     string
	Database string
	// Credentials are nil when no-auth cmd-line flag is supplied.
	Credentials        map[string][]byte // TODO: encapsulate credentials and mutex for safety
	CredentialsRWMutex sync.RWMutex
	// CredentialsPath is nil when no-auth is supplied.
	CredentialsPath string
}

func main() {
	if err := parseFlags(); err != nil {
		log.Fatalf("Error while parsing flags: %v", err)
		return
	}

	// Reload credentials when you receive SIGHUP
	sighupChan := make(chan os.Signal, 1)
	signal.Notify(sighupChan, syscall.SIGHUP)
	go func() {
		for range sighupChan {
			opts.CredentialsRWMutex.Lock()
			if opts.Credentials == nil {
				log.Println("Ignoring SIGHUP since `no-auth` was supplied")
				continue
			}

			opts.Credentials = make(map[string][]byte) // Clear opts.Credentials
			opts.CredentialsRWMutex.Unlock()
			if err := loadCred(opts.CredentialsPath); err != nil { // Reload credentials
				log.Printf("couldn't load credentials %v", err)
			}
		}
	}()

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

	database, err = persistence.MakeDatabase(opts.Database)
	if err != nil {
		log.Panicf("could not access to database %v", err)
	}

	decoder.IgnoreUnknownKeys(false)
	decoder.ZeroEmpty(true)

	log.Printf("magneticow is ready to serve on %s!", opts.Addr)
	err = http.ListenAndServe(opts.Addr, router)
	if err != nil {
		log.Printf("ListenAndServe error %v", err)
	}
}

// TODO: I think there is a standard lib. function for this
func respondError(w http.ResponseWriter, statusCode int, format string, a ...interface{}) {
	w.WriteHeader(statusCode)
	_, _ = w.Write([]byte(fmt.Sprintf(format, a...)))
}

func mustAsset(name string) []byte {
	data, err := fs.ReadFile(name)
	if err != nil {
		log.Panicf("Could NOT access the requested resource! THIS IS A BUG, PLEASE REPORT. %v", err)
	}
	return data
}

func parseFlags() error {
	var cmdFlags struct {
		Addr     string `short:"a" long:"addr"        description:"Address (host:port) to serve on"  default:":8080"`
		Database string `short:"d" long:"database"    description:"DSN of the database"`
		Cred     string `short:"c" long:"credentials" description:"Path to the credentials file"`
		NoAuth   bool   `          long:"no-auth"     description:"Disables authorisation"`
	}

	if _, err := flags.Parse(&cmdFlags); err != nil {
		return err
	}

	if cmdFlags.Cred != "" && cmdFlags.NoAuth {
		return fmt.Errorf("`credentials` and `no-auth` cannot be supplied together")
	}

	if cmdFlags.Cred == "" && !cmdFlags.NoAuth {
		return fmt.Errorf("you should supply either `credentials` or `no-auth`")
	}

	opts.Addr = cmdFlags.Addr

	if cmdFlags.Database == "" {
		opts.Database =
			"postgres://magnetico:magnetico@localhost:5432/magnetico?sslmode=disable"
	} else {
		opts.Database = cmdFlags.Database
	}

	if !cmdFlags.NoAuth {
		if cmdFlags.Cred != "" {
			opts.CredentialsPath = cmdFlags.Cred
		}

		opts.Credentials = make(map[string][]byte)
		if err := loadCred(opts.CredentialsPath); err != nil {
			return err
		}
	}

	return nil
}

func loadCred(cred string) error {
	file, err := os.Open(cred)
	if err != nil {
		return err
	}

	opts.CredentialsRWMutex.Lock()
	defer opts.CredentialsRWMutex.Unlock()

	reader := bufio.NewReader(file)
	for lineno := 1; true; lineno++ {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return errors.New("Error while reading line " + strconv.Itoa(lineno) + " " + err.Error())
		}

		line = line[:len(line)-1] // strip '\n'

		/* The following regex checks if the line satisfies the following conditions:
		 *
		 * <USERNAME>:<BCRYPT HASH>
		 *
		 * where
		 *     <USERNAME> must start with a small-case a-z character, might contain non-consecutive
		 *   underscores in-between, and consists of small-case a-z characters and digits 0-9.
		 *
		 *     <BCRYPT HASH> is the output of the well-known bcrypt function.
		 */
		re := regexp.MustCompile(`^[a-z](?:_?[a-z0-9])*:\$2[aby]?\$\d{1,2}\$[./A-Za-z0-9]{53}$`)
		if !re.Match(line) {
			return fmt.Errorf("on line %d: format should be: <USERNAME>:<BCRYPT HASH>, instead got: %s", lineno, line)
		}

		tokens := bytes.Split(line, []byte(":"))
		opts.Credentials[string(tokens[0])] = tokens[1]
	}

	return nil
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
		if opts.Credentials == nil { // --no-auth is supplied by the user.
			handler(w, r)
			return
		}

		username, password, ok := r.BasicAuth()
		if !ok { // No credentials provided
			authenticate(w)
			return
		}

		opts.CredentialsRWMutex.RLock()
		hashedPassword, ok := opts.Credentials[username]
		opts.CredentialsRWMutex.RUnlock()
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

func authenticate(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="magneticow"`)
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte("Unauthorised.\n"))
}
