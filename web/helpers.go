package web

import (
	"fmt"
	"log"
	"net/http"

	"github.com/tgragnato/magnetico/persistence"
)

func StartWeb(address string, cred map[string][]byte, db persistence.Database) {
	credentials = cred
	database = db
	log.Printf("magnetico is ready to serve on %s!", address)
	err := http.ListenAndServe(address, makeRouter())
	if err != nil {
		log.Fatalf("ListenAndServe error %v", err)
	}
}

func handlerError(err error, w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = w.Write([]byte(err.Error()))
}

// TODO: I think there is a standard lib. function for this
func respondError(w http.ResponseWriter, statusCode int, format string, a ...interface{}) {
	w.WriteHeader(statusCode)
	_, _ = w.Write([]byte(fmt.Sprintf(format, a...)))
}

func UpdateCredentials(newCredentials map[string][]byte) {
	credentialsRWMutex.Lock()
	credentials = newCredentials
	credentialsRWMutex.Unlock()
}
