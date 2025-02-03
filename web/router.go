package web

import (
	"context"
	"embed"
	"log"
	"net/http"
	"time"

	"tgragnato.it/magnetico/persistence"
	"tgragnato.it/magnetico/stats"
	"tgragnato.it/magnetico/types/infohash"
	infohash_v2 "tgragnato.it/magnetico/types/infohash-v2"
)

var (
	//go:embed static/**
	static   embed.FS
	database persistence.Database
)

type InfohashKeyType string

const (
	ContentType     string          = "Content-Type"
	ContentTypeJson string          = "application/json; charset=utf-8"
	InfohashKey     InfohashKeyType = "infohash"
)

func StartWeb(address string, cred map[string][]byte, db persistence.Database) {
	credentials = cred
	database = db
	log.Printf("magnetico is ready to serve on %s!\n", address)
	server := &http.Server{
		Addr:         address,
		Handler:      makeRouter(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("ListenAndServe error %s\n", err.Error())
	}
}

func makeRouter() *http.ServeMux {
	router := http.NewServeMux()

	router.HandleFunc("/", BasicAuth(rootHandler))

	staticFS := http.FS(static)
	router.HandleFunc("GET /static/", BasicAuth(
		http.StripPrefix("/", http.FileServer(staticFS)).ServeHTTP,
	))

	router.HandleFunc("/metrics", BasicAuth(stats.MakePrometheusHandler()))

	router.HandleFunc("GET /api/v0.1/statistics", BasicAuth(apiStatistics))
	router.HandleFunc("GET /api/v0.1/torrents", BasicAuth(apiTorrents))
	router.HandleFunc("GET /api/v0.1/torrentstotal", BasicAuth(apiTorrentsTotal))
	router.HandleFunc("GET /api/v0.1/torrents/{infohash}", BasicAuth(infohashMiddleware(apiTorrent)))
	router.HandleFunc("GET /api/v0.1/torrents/{infohash}/filelist", BasicAuth(infohashMiddleware(apiFileList)))

	router.HandleFunc("GET /robots.txt", BasicAuth(robotsHandler))
	router.HandleFunc("GET /feed", BasicAuth(feedHandler))
	router.HandleFunc("GET /statistics", BasicAuth(statisticsHandler))
	router.HandleFunc("GET /torrents/", BasicAuth(torrentsInfohashHandler))
	router.HandleFunc("GET /torrents", BasicAuth(torrentsHandler))

	return router
}

func robotsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(ContentType, "text/plain")
	w.Write([]byte("User-agent: *\nDisallow: /\n"))
}

func infohashMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		infohashHex := r.PathValue("infohash")

		var infohashBytes []byte
		if h1 := infohash.FromHexString(infohashHex); !h1.IsZero() {
			infohashBytes = h1.Bytes()
		} else if h2 := infohash_v2.FromHexString(infohashHex); !h2.IsZero() {
			infohashBytes = h2.Bytes()
		} else {
			http.Error(w, "Couldn't decode infohash", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, InfohashKey, infohashBytes)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
