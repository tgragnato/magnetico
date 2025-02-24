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
	ContentTypeHtml string          = "text/html; charset=utf-8"
	InfohashKey     InfohashKeyType = "infohash"
)

func StartWeb(address string, timeout uint, cred map[string][]byte, db persistence.Database) {
	credentials = cred
	database = db
	log.Printf("magnetico is ready to serve on %s!\n", address)
	timeoutDuration := time.Duration(timeout) * time.Second
	server := &http.Server{
		Addr:              address,
		Handler:           makeRouter(),
		ReadTimeout:       timeoutDuration,
		ReadHeaderTimeout: timeoutDuration,
		WriteTimeout:      timeoutDuration,
		IdleTimeout:       timeoutDuration,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("ListenAndServe error %s\n", err.Error())
	}
}

func middlewares(next http.HandlerFunc) http.HandlerFunc {
	return compressMiddleware(basicAuth(next))
}

func makeRouter() *http.ServeMux {
	router := http.NewServeMux()

	router.HandleFunc("/", middlewares(rootHandler))

	staticFS := http.FS(static)
	router.HandleFunc("GET /static/", middlewares(
		http.StripPrefix("/", http.FileServer(staticFS)).ServeHTTP,
	))

	router.HandleFunc("/metrics", middlewares(stats.MakePrometheusHandler()))

	router.HandleFunc("GET /api/v0.1/statistics", middlewares(apiStatistics))
	router.HandleFunc("GET /api/v0.1/torrents", middlewares(apiTorrents))
	router.HandleFunc("GET /api/v0.1/torrentstotal", middlewares(apiTorrentsTotal))
	router.HandleFunc("GET /api/v0.1/torrents/{infohash}", middlewares(infohashMiddleware(apiTorrent)))
	router.HandleFunc("GET /api/v0.1/torrents/{infohash}/filelist", middlewares(infohashMiddleware(apiFileList)))

	router.HandleFunc("GET /robots.txt", middlewares(robotsHandler))
	router.HandleFunc("GET /feed", middlewares(feedHandler))
	router.HandleFunc("GET /statistics", middlewares(statisticsHandler))
	router.HandleFunc("GET /torrents/", middlewares(torrentsInfohashHandler))
	router.HandleFunc("GET /torrents", middlewares(torrentsHandler))

	return router
}

func robotsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(ContentType, "text/plain")
	if _, err := w.Write([]byte("User-agent: *\nDisallow: /\n")); err != nil {
		http.Error(w, "Failed to write response: "+err.Error(), http.StatusInternalServerError)
	}
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
