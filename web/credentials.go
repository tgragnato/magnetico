package web

import (
	"net/http"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

var (
	credentials        map[string][]byte
	credentialsRWMutex sync.RWMutex
)

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
func basicAuth(handler http.HandlerFunc) http.HandlerFunc {
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

func authenticate(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="magneticow"`)
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte("Unauthorised.\n"))
}

func UpdateCredentials(newCredentials map[string][]byte) {
	credentialsRWMutex.Lock()
	credentials = newCredentials
	credentialsRWMutex.Unlock()
}
